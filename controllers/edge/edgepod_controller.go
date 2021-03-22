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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	edgev1alpha1 "github.com/pbowden/edge-deploy/apis/edge/v1alpha1"
)

// EdgePodReconciler reconciles a EdgePod object
type EdgePodReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=edge.pete.dev,resources=edgepods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=edge.pete.dev,resources=edgepods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=edge.pete.dev,resources=edgepods/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EdgePod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *EdgePodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("edgepod", req.NamespacedName)

	edgePod := &edgev1alpha1.EdgePod{}
	err := r.Get(ctx, req.NamespacedName, edgePod)

	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("EdgePod resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get EdgePod")
		return ctrl.Result{}, err
	}

	podspec := &edgev1alpha1.InternalPodspec{
		ApiVersion: "v1",
		Kind:       "Pod",
		//ObjectMeta: edgePod.ObjectMeta,
		Spec: edgePod.Spec,
	}

	if !reflect.DeepEqual(podspec, edgePod.Podspec) {
		log.Info("Podspec has changed, updating internal field", "PodSpec.Namespace", edgePod.Namespace, "edgePod.Name", edgePod.Name)
		edgePod.Podspec = podspec
		err = r.Update(ctx, edgePod)
		if err != nil {
			log.Error(err, "Failed to update EdgePod status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgePodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&edgev1alpha1.EdgePod{}).
		Complete(r)
}
