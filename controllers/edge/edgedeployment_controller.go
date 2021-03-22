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

	edgev1alpha1 "github.com/pbowden/edge-deploy/apis/edge/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

	edgedeployment := &edgev1alpha1.EdgeDeployment{}
	err := r.Get(ctx, req.NamespacedName, edgedeployment)
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

	//requeue := false

	// Get a list of all EdgePods affiliated with this deployment
	edgePodList := &edgev1alpha1.EdgePodList{}
	listOpts := []client.ListOption{
		client.InNamespace(edgedeployment.Namespace),
		client.MatchingLabels(map[string]string{"deploymentName": edgedeployment.Name}),
	}
	if err = r.List(ctx, edgePodList, listOpts...); err != nil {
		log.Error(err, "Failed to list edgePods", "edgedeployment.Namespace", edgedeployment.Namespace, "edgedeployment.Name", edgedeployment.Name)
		return ctrl.Result{}, err
	}
	// Sort the list
	sort.Slice(edgePodList.Items, func(i, j int) bool {
		return edgePodList.Items[i].Name < edgePodList.Items[j].Name
	})
	// Compare the lists, updating or deleting as needed
	listEdgeNodes := edgedeployment.Spec.EdgeNodes
	sort.Slice(listEdgeNodes, func(i, j int) bool {
		return listEdgeNodes[i] < listEdgeNodes[j]
	})

	i, j := 0, 0
	for i < len(listEdgeNodes) && j < len(edgePodList.Items) {
		// Is there an edgePod for this Deployment / Edge Node
		if listEdgeNodes[i] == edgePodList.Items[j].Name {
			// Is it up to date?
			//TODO: compare
			i, j = i+1, j+1

		} else if listEdgeNodes[i] > edgePodList.Items[j].Name {
			// Do we need to delete the mismatch?
			// Yes
			// Delete edgePodList.Items[j]
			j++
		} else {
			// No existing edgePod for Deployment / Edge Node
			// todo: create
			i++
		}
	}

	for ; i < len(listEdgeNodes); i++ {
		//Create remaining EdgePods
		//TODO: Implement
	}

	for ; j < len(edgePodList.Items); j++){
		// Delete remaining uneeded EdgePods
		//TODO: Implement
	}

	foundPodSpec := &edgev1alpha1.EdgePod{}
	// Loop through each edge node to see if it has a PodSpec
	for _, edgeNodeName := range edgedeployment.Spec.EdgeNodes {
		log.Info("EdgeNode Loop", "Name", edgeNodeName)
		err = r.Get(ctx, types.NamespacedName{Name: getPodSpecName(edgeNodeName, edgedeployment.Name), Namespace: edgedeployment.Namespace}, foundPodSpec)
		// If it doesn't have one, create it
		if err != nil && errors.IsNotFound(err) {
			podSpec := r.podSpecForDeployment(edgedeployment, edgeNodeName)
			log.Info("Creating a new PodSpec", "PodSpec.Namespace", podSpec.Namespace, "PodSpec.Name", podSpec.Name)
			err = r.Create(ctx, podSpec)
			if err != nil {
				log.Error(err, "Failed to create new PodSpec", "PodSpec.Namespace", podSpec.Namespace, "PodSpec.Name", podSpec.Name)
				return ctrl.Result{}, err
			}
			// we created an object, we should requeue
			return ctrl.Result{Requeue: true}, nil

			//			requeue = true
		} else if err != nil {
			log.Error(err, "Failed to get PodSpec")
			return ctrl.Result{}, err
		} else {
			// Found a podspec, does it match the deployment spec?
			edgePod := r.podSpecForDeployment(edgedeployment, edgeNodeName)
			if !reflect.DeepEqual(foundPodSpec.Spec, edgePod.Spec) {
				log.Info("EdgePod doesn't match Deployment. Updating EdgePod ", "PodSpec.Namespace ", foundPodSpec.Namespace, "PodSpec.Name ", foundPodSpec.Name)
				foundPodSpec.Spec = edgePod.Spec
				err = r.Update(ctx, foundPodSpec)
				if err != nil {
					log.Error(err, "Failed to update EdgePod status")
					return ctrl.Result{}, err
				}
				return ctrl.Result{Requeue: true}, nil

				//				requeue = true
			}
			//return ctrl.Result{}, err
		}
	}
	// PodSpecs created/updated successfully, return and requeue
	/* 	if requeue {
		return ctrl.Result{Requeue: true}, nil
	} */

	return ctrl.Result{}, nil
}

func (r *DeploymentReconciler) podSpecForDeployment(d *edgev1alpha1.EdgeDeployment, edgeNodeName string) *edgev1alpha1.EdgePod {
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

func getPodSpecName(edgeNode string, deploymentName string) string {
	return edgeNode + "-" + deploymentName
}

func labelsForEdgeDeployment(deploymentName string, edgeNodeName string) map[string]string {
	return map[string]string{"deploymentName": deploymentName, "edgeNode": edgeNodeName}
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&edgev1alpha1.EdgeDeployment{}).
		Complete(r)
}
