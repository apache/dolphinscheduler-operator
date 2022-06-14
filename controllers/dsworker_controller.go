/*
Copyright 2022.

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

package controllers

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	dsv1alpha1 "dolphinscheduler-operator/api/v1alpha1"
)

// DSWorkerReconciler reconciles a DSWorker object
type DSWorkerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

var (
	workerLogger = ctrl.Log.WithName("DSWorker-controller")
)

//+kubebuilder:rbac:groups=ds.apache.dolphinscheduler.dev,resources=dsworkers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ds.apache.dolphinscheduler.dev,resources=dsworkers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ds.apache.dolphinscheduler.dev,resources=dsworkers/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;create;delete;list;watch
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the DSWorker object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *DSWorkerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	workerLogger.Info("dmWorker start reconcile logic")
	defer workerLogger.Info("dmWorker Reconcile end ---------------------------------------------")

	cluster := &dsv1alpha1.DSWorker{}

	if err := r.Client.Get(ctx, req.NamespacedName, cluster); err != nil {
		if errors.IsNotFound(err) {
			r.recorder.Event(cluster, corev1.EventTypeWarning, "dmWorker is not Found", "dmWorker is not Found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	desired := cluster.DeepCopy()

	// Handler finalizer
	// examine DeletionTimestamp to determine if object is under deletion
	if cluster.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(desired, dsv1alpha1.FinalizerName) {
			controllerutil.AddFinalizer(desired, dsv1alpha1.FinalizerName)
			if err := r.Update(ctx, desired); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(desired, dsv1alpha1.FinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.ensureDSWorkerDeleted(ctx, cluster); err != nil {
				return ctrl.Result{}, err
			}
			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(desired, dsv1alpha1.FinalizerName)
			if err := r.Update(ctx, desired); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// If dsworker-cluster is paused, we do nothing on things changed.
	// Until dsworker-cluster is un-paused, we will reconcile to the dsworker state of that point.
	if cluster.Spec.Paused {
		workerLogger.Info("ds-worker control has been paused: ", "ds-worker-name", cluster.Name)
		desired.Status.ControlPaused = true
		if err := r.Status().Patch(ctx, desired, client.MergeFrom(cluster)); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			} else {
				masterLogger.Error(err, "unexpected error when worker update status in paused")
				return ctrl.Result{}, err
			}
		}
		r.recorder.Event(cluster, corev1.EventTypeNormal, "the spec status is paused", "do nothing")
		return ctrl.Result{}, nil
	}

	// 1. First time we see the ds-worker-cluster, initialize it
	if cluster.Status.Phase == dsv1alpha1.DsPhaseNone {
		if desired.Status.Selector == "" {
			selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: LabelForWorkerPod()})
			if err != nil {
				masterLogger.Error(err, "Error retrieving selector labels")
				return reconcile.Result{}, err
			}
			desired.Status.Selector = selector.String()
		}
		desired.Status.Phase = dsv1alpha1.DsPhaseCreating
		workerLogger.Info("phase had been changed from  none ---> creating")
		if err := r.Client.Status().Patch(ctx, desired, client.MergeFrom(cluster)); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, err
			} else {
				masterLogger.Error(err, "unexpected error when worker update status in creating")
				return ctrl.Result{}, err
			}
		}
	}

	// 3. Ensure bootstrapped, we will block here util cluster is up and healthy
	workerLogger.Info("Ensuring cluster members")
	if requeue, err := r.ensureMembers(ctx, cluster); requeue {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	// 4. Ensure cluster scaled
	workerLogger.Info("Ensuring cluster scaled")
	if requeue, err := r.ensureScaled(ctx, cluster); requeue {
		return ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, err
	}

	// .5 Ensure cluster upgraded
	workerLogger.Info("Ensuring cluster upgraded")
	if requeue, err := r.ensureUpgraded(ctx, cluster); requeue {
		return ctrl.Result{Requeue: true}, err
	}

	desired.Status.Phase = dsv1alpha1.DsPhaseFinished
	if err := r.Status().Patch(ctx, desired, client.MergeFrom(cluster)); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		} else {
			masterLogger.Error(err, "unexpected error when worker update status in finished")
			return ctrl.Result{}, err
		}
	}

	workerLogger.Info("******************************************************")
	desired.Status.Phase = dsv1alpha1.DsPhaseNone
	return ctrl.Result{Requeue: false}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DSWorkerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("worker-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&dsv1alpha1.DSWorker{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

func (r *DSWorkerReconciler) ensureMembers(ctx context.Context, cluster *dsv1alpha1.DSWorker) (bool, error) {

	pms, err := r.podMemberSet(ctx, cluster)
	if err != nil {
		return true, err
	}
	if len(pms) > 0 {
		return !allMembersHealth(pms), nil
	} else {
		return false, nil
	}

}

func (r *DSWorkerReconciler) ensureScaled(ctx context.Context, cluster *dsv1alpha1.DSWorker) (bool, error) {
	// Get current members in this cluster
	ms, err := r.podMemberSet(ctx, cluster)
	if err != nil {
		return true, err
	}

	// Scale up
	if len(ms) < cluster.Spec.Replicas {
		err = r.createMember(ctx, cluster)
		if err != nil {
			r.recorder.Event(cluster, corev1.EventTypeWarning, "cannot create the new ds-dsworker pod", "the ds-dsworker pod had been created failed")
			return true, err
		}
		return true, err
	}

	// Scale down
	if len(ms) > cluster.Spec.Replicas {
		pod := &corev1.Pod{}
		member := ms.PickOne()
		pod.SetName(member.Name)
		pod.SetNamespace(member.Namespace)
		err = r.deleteMember(ctx, pod, cluster)
		if err != nil {
			return true, err
		}
		return true, err
	}

	return false, nil
}

func (r *DSWorkerReconciler) ensureUpgraded(ctx context.Context, cluster *dsv1alpha1.DSWorker) (bool, error) {

	ms, err := r.podMemberSet(ctx, cluster)
	if err != nil {
		return false, err
	}

	for _, memset := range ms {
		if memset.Version != cluster.Spec.Version {
			pod := &corev1.Pod{}
			pod.SetName(memset.Name)
			pod.SetNamespace(memset.Namespace)
			if err := r.deleteMember(ctx, pod, cluster); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func (r *DSWorkerReconciler) ensureDSWorkerDeleted(ctx context.Context, cluster *dsv1alpha1.DSWorker) error {
	if err := r.Client.Delete(ctx, cluster, client.PropagationPolicy(metav1.DeletePropagationOrphan)); err != nil {
		return err
	}
	return nil
}

func (r *DSWorkerReconciler) createMember(ctx context.Context, cluster *dsv1alpha1.DSWorker) error {
	workerLogger.Info("Starting add new member to cluster", "cluster", cluster.Name)
	defer workerLogger.Info("End add new member to cluster", "cluster", cluster.Name)

	// New Pod
	pod, err := r.newDSWorkerPod(ctx, cluster)
	if err != nil {
		return err
	}

	// Create pod
	if err = r.Client.Create(ctx, pod); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	desired := cluster.DeepCopy()
	desired.Spec.Replicas += 1
	if err := r.Status().Patch(ctx, desired, client.MergeFrom(cluster)); err != nil {
		return err
	}
	return nil
}

func (r *DSWorkerReconciler) deleteMember(ctx context.Context, pod *corev1.Pod, cluster *dsv1alpha1.DSWorker) error {

	workerLogger.Info("begin delete pod", "pod name", pod.Name)
	if err := r.Client.Delete(ctx, pod); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	desired := cluster.DeepCopy()
	desired.Spec.Replicas -= 1
	if err := r.Status().Patch(ctx, desired, client.MergeFrom(cluster)); err != nil {
		return err
	}
	return nil
}
