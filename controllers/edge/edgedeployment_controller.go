/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package edge

import (
	"context"
	"reflect"
	"sort"

	"github.com/go-logr/logr"

	edgev1alpha1 "github.com/petebowden/edge-deploy/apis/edge/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeploymentReconciler reconciles a EdgeDeployment object
type DeploymentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=edge.pete.dev,resources=edgedeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=edge.pete.dev,resources=edgedeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=edge.pete.dev,resources=edgedeployments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EdgeDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("edgedeployment", req.NamespacedName)

	log.Info("Reconcile", "request Namespace", req.Namespace, "request Name", req.Name)

	foundEdgeDeployment := &edgev1alpha1.EdgeDeployment{}
	err := r.Get(ctx, req.NamespacedName, foundEdgeDeployment)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("EdgeDeployment resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get EdgeDeployment")
		return ctrl.Result{}, err
	}

	// Get a list of all EdgePods affiliated with this deployment
	foundEdgePodList := &edgev1alpha1.EdgePodList{}
	listOpts := []client.ListOption{
		client.InNamespace(foundEdgeDeployment.Namespace),
		client.MatchingLabels(map[string]string{"deploymentName": foundEdgeDeployment.Name}),
	}
	if err = r.List(ctx, foundEdgePodList, listOpts...); err != nil {
		log.Error(err, "Failed to list edgePods", "edgedeployment.Namespace", foundEdgeDeployment.Namespace, "edgedeployment.Name", foundEdgeDeployment.Name)
		return ctrl.Result{}, err
	}
	// Sort the lists
	sort.Slice(foundEdgePodList.Items, func(i, j int) bool {
		return foundEdgePodList.Items[i].Name > foundEdgePodList.Items[j].Name
	})

	listEdgeNodes := foundEdgeDeployment.Spec.EdgeNodes
	sort.Slice(listEdgeNodes, func(i, j int) bool {
		return listEdgeNodes[i] > listEdgeNodes[j]
	})

	// Var to hold if we need to requeue after creating objects
	requeue := false

	i, j := 0, 0
	// Compare the lists, updating or deleting as needed
	for i < len(listEdgeNodes) && j < len(foundEdgePodList.Items) {

		// Is there an edgePod for this Deployment / Edge Node
		edgeNodePodName := getPodSpecName(listEdgeNodes[i], foundEdgeDeployment.Name)
		if edgeNodePodName == foundEdgePodList.Items[j].Name {

			// Does EdgePod Spec match the deployment Spec?
			if !reflect.DeepEqual(foundEdgePodList.Items[j].Spec, foundEdgeDeployment.Spec.Template.Spec) {
				log.Info("Updating EdgePod", "EdgePod.Name", foundEdgePodList.Items[j].Name)
				// Doesn't match, update
				foundEdgePodList.Items[j].Spec = foundEdgeDeployment.Spec.Template.Spec
				err = r.Update(ctx, &foundEdgePodList.Items[j])
				if err != nil {
					log.Error(err, "Failed to update EdgePod status")
					return ctrl.Result{}, err
				}
			}
			//increment both pointers
			i, j = i+1, j+1

		} else if edgeNodePodName < foundEdgePodList.Items[j].Name {
			// Do we need to delete an Edge Pod?
			// Yes
			err = r.Delete(ctx, &foundEdgePodList.Items[j])
			if err != nil {
				log.Error(err, "Failed to delete EdgePod status")
				return ctrl.Result{}, err
			}
			j++
		} else {
			// No, create new edgePod for Deployment / Edge Node
			err := r.createNewEdgePod(foundEdgeDeployment, listEdgeNodes[i], ctx)
			if err != nil {
				return ctrl.Result{}, err
			}
			requeue = true
			i++
		}
	}

	// Create remaining EdgePods
	for ; i < len(listEdgeNodes); i++ {
		err := r.createNewEdgePod(foundEdgeDeployment, listEdgeNodes[i], ctx)
		if err != nil {
			return ctrl.Result{}, err
		}
		requeue = true
		i++
	}

	// Delete remaining uneeded EdgePods
	for ; j < len(foundEdgePodList.Items); j++ {
		err = r.Delete(ctx, &foundEdgePodList.Items[j])
		if err != nil {
			log.Error(err, "Failed to delete EdgePod status")
			return ctrl.Result{}, err
		}
		j++
	}

	if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

func (r *DeploymentReconciler) createNewEdgePod(d *edgev1alpha1.EdgeDeployment, edgeNodeName string, ctx context.Context) error {
	podSpec := r.edgePodForDeployment(d, edgeNodeName)

	r.Log.Info("Creating a new PodSpec", "PodSpec.Namespace", podSpec.Namespace, "PodSpec.Name", podSpec.Name)
	err := r.Create(ctx, podSpec)
	if err != nil {
		r.Log.Error(err, "Failed to create new PodSpec", "PodSpec.Namespace", podSpec.Namespace, "PodSpec.Name", podSpec.Name)
		return err
	}
	return nil
}

func (r *DeploymentReconciler) edgePodForDeployment(d *edgev1alpha1.EdgeDeployment, edgeNodeName string) *edgev1alpha1.EdgePod {
	labels := labelsForEdgeDeployment(d.Name, edgeNodeName)
	podSpecName := getPodSpecName(edgeNodeName, d.Name)
	podSpec := &edgev1alpha1.EdgePod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podSpecName,
			Namespace: d.Namespace,
			Labels:    labels,
		},
		//Podspec:    edgev1alpha1.InternalPodspec{},
		EdgeTarget: edgeNodeName,
		Spec: edgev1alpha1.EdgePodSpec{
			Containers: d.Spec.Template.Spec.Containers,
		},
	}
	ctrl.SetControllerReference(d, podSpec, r.Scheme)
	return podSpec
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&edgev1alpha1.EdgeDeployment{}).
		Complete(r)
}

func getPodSpecName(edgeNode string, deploymentName string) string {
	return edgeNode + "-" + deploymentName
}

func labelsForEdgeDeployment(deploymentName string, edgeNodeName string) map[string]string {
	return map[string]string{"deploymentName": deploymentName, "edgeNode": edgeNodeName}
}
