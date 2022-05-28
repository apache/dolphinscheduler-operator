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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sync"
	"time"

	dsv1alpha1 "dolphinscheduler-operator/api/v1alpha1"
)

const (
	dsMasterLabel  = "ds-master"
	dsServiceLabel = "ds-operator-service"
	dsServiceName  = "ds-operator-service"
)

var (
	logger = ctrl.Log.WithName("DSMaster-controller")
)

// DSMasterReconciler reconciles a DSMaster object
type DSMasterReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	clusters sync.Map
	resyncCh chan event.GenericEvent
}

//+kubebuilder:rbac:groups=ds.apache.dolphinscheduler.dev,resources=dsmasters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ds.apache.dolphinscheduler.dev,resources=dsmasters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ds.apache.dolphinscheduler.dev,resources=dsmasters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the DSMaster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *DSMasterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger.Info("dmMaster start reconcile logic")
	defer logger.Info("dmMaster Reconcile end ---------------------------------------------")

	cluster := &dsv1alpha1.DSMaster{}

	if err := r.Client.Get(ctx, req.NamespacedName, cluster); err != nil {
		if errors.IsNotFound(err) {
			r.Recorder.Event(cluster, corev1.EventTypeWarning, "dsMaster is not Found", "dsMaster is not Found")
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
			if err := r.ensureDSMasterDeleted(ctx, cluster); err != nil {
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

	// If dsmaster-cluster is paused, we do nothing on things changed.
	// Until dsmaster-cluster is un-paused, we will reconcile to the the state of that point.
	if cluster.Spec.Paused {
		logger.Info("ds-master control has been paused: ", "ds-master-name", cluster.Name)
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
		logger.Info("phase had been changed from  none ---> creating")
		err := r.Client.Status().Patch(ctx, desired, client.MergeFrom(cluster))
		return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, err
	}

	//2 ensure the headless service
	logger.Info("Ensuring cluster service")

	if err := r.ensureMasterService(ctx, cluster); err != nil {
		return ctrl.Result{}, err
	}

	// 3. Ensure bootstrapped, we will block here util cluster is up and healthy
	logger.Info("Ensuring cluster members")
	if requeue, err := r.ensureMembers(ctx, cluster); requeue {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	// 4. Ensure cluster scaled
	logger.Info("Ensuring cluster scaled")
	if requeue, err := r.ensureScaled(ctx, cluster); requeue {
		return ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, err
	}

	// .5 Ensure cluster upgraded
	logger.Info("Ensuring cluster upgraded")
	if requeue, err := r.ensureUpgraded(ctx, cluster); requeue {
		return ctrl.Result{Requeue: true}, err
	}

	desired.Status.Phase = dsv1alpha1.DsPhaseFinished
	if err := r.Status().Patch(ctx, desired, client.MergeFrom(cluster)); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("******************************************************")
	desired.Status.Phase = dsv1alpha1.DsPhaseNone
	if err := r.Update(ctx, desired); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: false}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DSMasterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.clusters = sync.Map{}
	r.resyncCh = make(chan event.GenericEvent)
	r.Recorder = mgr.GetEventRecorderFor("master-controller")

	filter := &Predicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&dsv1alpha1.DSMaster{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Watches(&source.Channel{Source: r.resyncCh}, &handler.EnqueueRequestForObject{}).
		// or use WithEventFilter()
		WithEventFilter(filter).
		Complete(r)
}

func (r *DSMasterReconciler) ensureMembers(ctx context.Context, cluster *dsv1alpha1.DSMaster) (bool, error) {

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

func (r *DSMasterReconciler) ensureScaled(ctx context.Context, cluster *dsv1alpha1.DSMaster) (bool, error) {
	// Get current members in this cluster
	ms, err := r.podMemberSet(ctx, cluster)
	if err != nil {
		return true, err
	}

	// Scale up
	if len(ms) < cluster.Spec.Replicas {
		err = r.createMember(ctx, cluster)
		if err != nil {
			r.Recorder.Event(cluster, corev1.EventTypeWarning, "cannot create the new ds-master pod", "the ds-master pod had been created failed")
			return true, err
		}
		// Cluster modified, next reconcile will enter r.ensureMembers()
		return true, err
	}

	// Scale down
	if len(ms) > cluster.Spec.Replicas {
		pod := &corev1.Pod{}
		member := ms.PickOne()
		pod.SetName(member.Name)
		pod.SetNamespace(member.Namespace)
		err = r.deletePod(ctx, pod)
		if err != nil {
			return true, err
		}
		return true, err
	}

	return false, nil
}

func (r *DSMasterReconciler) createMember(ctx context.Context, cluster *dsv1alpha1.DSMaster) error {
	logger.Info("Starting add new member to cluster", "cluster", cluster.Name)
	defer logger.Info("End add new member to cluster", "cluster", cluster.Name)

	// New Pod
	pod, err := r.newDSMasterPod(ctx, cluster)
	if err != nil {
		return err
	}

	// Create pod
	if err = r.Client.Create(ctx, pod); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (r *DSMasterReconciler) deletePod(ctx context.Context, pod *corev1.Pod) error {

	logger.Info("begin delete pod", "pod name", pod.Name)
	if err := r.Client.Delete(ctx, pod); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

func (r *DSMasterReconciler) ensureUpgraded(ctx context.Context, cluster *dsv1alpha1.DSMaster) (bool, error) {

	ms, err := r.podMemberSet(ctx, cluster)
	if err != nil {
		return false, err
	}

	logger.Info("cluster.Spec.Version", "cluster.Spec.Version", cluster.Spec.Version)
	for _, memset := range ms {
		if memset.Version != cluster.Spec.Version {
			pod := &corev1.Pod{}
			pod.SetName(memset.Name)
			pod.SetNamespace(memset.Namespace)
			if err := r.deletePod(ctx, pod); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func getNeedUpgradePods(ctx context.Context, cli *kubernetes.Clientset, cluster *dsv1alpha1.DSMaster) (*corev1.PodList, error) {
	podSelector, err := labels.NewRequirement(dsv1alpha1.DsVersionLabel, selection.NotIn, []string{cluster.Spec.Version})
	if err != nil {
		return nil, err
	}
	podAppSelect, err := labels.NewRequirement(dsv1alpha1.DsAppName, selection.Equals, []string{dsMasterLabel})
	if err != nil {
		return nil, err
	}
	selector := labels.NewSelector()
	selector = selector.Add(*podSelector).Add(*podAppSelect)
	podListOptions := metav1.ListOptions{
		LabelSelector: selector.String(),
	}
	return cli.CoreV1().Pods(cluster.Namespace).List(ctx, podListOptions)
}

func (r *DSMasterReconciler) ensureMasterService(ctx context.Context, cluster *dsv1alpha1.DSMaster) error {

	// 1. Client service
	service := &corev1.Service{}
	namespacedName := types.NamespacedName{Namespace: cluster.Namespace, Name: dsv1alpha1.DsServiceLabelValue}
	if err := r.Client.Get(ctx, namespacedName, service); err != nil {
		// Local cache not found
		logger.Info("get service error")
		if apierrors.IsNotFound(err) {
			service = createMasterService(cluster)
			if err := controllerutil.SetControllerReference(cluster, service, r.Scheme); err != nil {
				logger.Info("create service error")
				return err
			}
			// Remote may already exist, so we will return err, for the next time, this code will not execute
			if err := r.Client.Create(ctx, service); err != nil {
				logger.Info("create service error1")
				return err
			}
			logger.Info("the headless service had been created")
		}
	}
	return nil
}

type Predicate struct{}

// Create will be trigger when object created or controller restart
// first time see the object
func (r *Predicate) Create(evt event.CreateEvent) bool {
	switch evt.Object.(type) {
	case *dsv1alpha1.DSMaster:
		return true
	}
	return false
}

func (r *Predicate) Update(evt event.UpdateEvent) bool {
	switch evt.ObjectNew.(type) {
	case *dsv1alpha1.DSMaster:
		oldC := evt.ObjectOld.(*dsv1alpha1.DSMaster)
		newC := evt.ObjectNew.(*dsv1alpha1.DSMaster)

		logger.V(5).Info("Running update filter",
			"Old size", oldC.Spec.Replicas,
			"New size", newC.Spec.Replicas,
			"old paused", oldC.Spec.Paused,
			"new paused", newC.Spec.Paused,
			"old object deletion", !oldC.ObjectMeta.DeletionTimestamp.IsZero(),
			"new object deletion", !newC.ObjectMeta.DeletionTimestamp.IsZero())

		// Only care about size, version and paused fields
		if oldC.Spec.Replicas != newC.Spec.Replicas {
			return true
		}

		if oldC.Spec.Paused != newC.Spec.Paused {
			return true
		}

		if oldC.Spec.Version != newC.Spec.Version {
			return true
		}

		// If cluster has been marked as deleted, check if we have remove our finalizer
		// If it has our finalizer, indicating our cleaning up works has not been done.
		if oldC.DeletionTimestamp.IsZero() && !newC.DeletionTimestamp.IsZero() {
			if controllerutil.ContainsFinalizer(newC, dsv1alpha1.FinalizerName) {
				return true
			}
		}
	}
	return false
}

func (r *Predicate) Delete(evt event.DeleteEvent) bool {
	switch evt.Object.(type) {
	case *dsv1alpha1.DSMaster:
		return true
	case *corev1.Pod:
		return true
	case *corev1.Service:
		return true
	}
	return false
}

func (r *Predicate) Generic(evt event.GenericEvent) bool {
	switch evt.Object.(type) {
	case *dsv1alpha1.DSMaster:
		return true
	}
	return false
}
