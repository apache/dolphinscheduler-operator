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
	dsv1alpha1 "dolphinscheduler-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

// DSAlertReconciler reconciles a DSAlert object
var (
	alertLogger = ctrl.Log.WithName("DSAlert-controller")
)

type DSAlertReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=ds.apache.dolphinscheduler.dev,resources=dsalerts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ds.apache.dolphinscheduler.dev,resources=dsalerts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ds.apache.dolphinscheduler.dev,resources=dsalerts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the DSAlert object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *DSAlertReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	alertLogger.Info("dmAlert start reconcile logic")
	defer alertLogger.Info("dmAlert Reconcile end ---------------------------------------------")

	cluster := &dsv1alpha1.DSAlert{}

	if err := r.Client.Get(ctx, req.NamespacedName, cluster); err != nil {
		if errors.IsNotFound(err) {
			r.Recorder.Event(cluster, corev1.EventTypeWarning, "dsAlert is not Found", "dsAlert is not Found")
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
			if err := r.ensureDSAlertDeleted(ctx, cluster); err != nil {
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(desired, dsv1alpha1.FinalizerName)
			if err := r.Update(ctx, desired); err != nil {
				return ctrl.Result{}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	if cluster.Spec.Paused {
		alertLogger.Info("ds-alert control has been paused: ", "ds-alert-name", cluster.Name)
		desired.Status.ControlPaused = true
		if err := r.Status().Patch(ctx, desired, client.MergeFrom(cluster)); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Event(cluster, corev1.EventTypeNormal, "the spec status is paused", "do nothing")
		return ctrl.Result{}, nil
	}

	// 1. First time we see the ds-master-cluster, initialize it
	if cluster.Status.Phase == dsv1alpha1.DsPhaseNone {
		desired.Status.Phase = dsv1alpha1.DsPhaseCreating
		alertLogger.Info("phase had been changed from  none ---> creating")
		err := r.Client.Status().Patch(ctx, desired, client.MergeFrom(cluster))
		return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, err
	}

	//2 ensure the alert service
	alertLogger.Info("Ensuring alert service")

	if err := r.ensureAlertService(ctx, cluster); err != nil {
		return ctrl.Result{Requeue: true}, nil
	}

	if requeue, err := r.ensureAlertDeployment(ctx, cluster); err != nil {
		return ctrl.Result{Requeue: false}, err
	} else {
		if !requeue {
			return ctrl.Result{Requeue: false}, nil
		}
	}

	alertLogger.Info("******************************************************")
	desired.Status.Phase = dsv1alpha1.DsPhaseNone
	if err := r.Update(ctx, desired); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: false}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DSAlertReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dsv1alpha1.DSAlert{}).
		Owns(&v1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

func (r *DSAlertReconciler) ensureDSAlertDeleted(ctx context.Context, DSAlert *dsv1alpha1.DSAlert) error {
	if err := r.Client.Delete(ctx, DSAlert, client.PropagationPolicy(metav1.DeletePropagationOrphan)); err != nil {
		return err
	}
	return nil
}

func (r *DSAlertReconciler) ensureAlertService(ctx context.Context, cluster *dsv1alpha1.DSAlert) error {
	// 1. Client service
	service := &corev1.Service{}
	namespacedName := types.NamespacedName{Namespace: cluster.Namespace, Name: dsv1alpha1.DsAlertServiceValue}
	if err := r.Client.Get(ctx, namespacedName, service); err != nil {
		// Local cache not found
		logger.Info("get service error")
		if apierrors.IsNotFound(err) {
			service = createAlertService(cluster)
			if err := controllerutil.SetControllerReference(cluster, service, r.Scheme); err != nil {
				logger.Info("create alert service error")
				return err
			}
			// Remote may already exist, so we will return err, for the next time, this code will not execute
			if err := r.Client.Create(ctx, service); err != nil {
				logger.Info("create alert service error1")
				return err
			}
			logger.Info("the alert service had been created")
		}
	}
	return nil
}

func (r *DSAlertReconciler) ensureAlertDeployment(ctx context.Context, cluster *dsv1alpha1.DSAlert) (bool, error) {
	deployment := &v1.Deployment{}
	deploymentNamespaceName := types.NamespacedName{Namespace: cluster.Namespace, Name: dsv1alpha1.DsAlertDeploymentValue}
	if err := r.Client.Get(ctx, deploymentNamespaceName, deployment); err != nil {
		if apierrors.IsNotFound(err) {
			deployment = createAlertDeployment(cluster)
		}
		if err := controllerutil.SetControllerReference(cluster, deployment, r.Scheme); err != nil {
			return true, err
		}
		if err := r.Client.Create(ctx, deployment); err == nil {
			return false, nil
		} else {
			return true, err
		}
	} else {
		err := r.updateAlertDeployment(ctx, deployment, cluster)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

//only notice the property of replicas  and image and version
func (r *DSAlertReconciler) updateAlertDeployment(ctx context.Context, deployment *v1.Deployment, cluster *dsv1alpha1.DSAlert) error {
	deployment.Spec.Replicas = int32Ptr(int32(cluster.Spec.Replicas))
	deployment.Spec.Template.Spec.Containers[0].Image = ImageName(cluster.Spec.Repository, cluster.Spec.Version)
	if err := r.Client.Update(ctx, deployment); err != nil {
		return err
	}
	return nil
}
