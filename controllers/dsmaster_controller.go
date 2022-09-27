// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package controllers

import (
	"context"
	"k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/rbac/v1"
	"time"

	dsv1alpha1 "dolphinscheduler-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
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
)

var (
	masterLogger = ctrl.Log.WithName("DSMaster-controller")
)

// DSMasterReconciler reconciles a DSMaster object
type DSMasterReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=ds.apache.dolphinscheduler.dev,resources=dsmasters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ds.apache.dolphinscheduler.dev,resources=dsmasters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ds.apache.dolphinscheduler.dev,resources=dsmasters/finalizers,verbs=update
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;create;delete;list;watch
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;create;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=role,verbs=get;list;create;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebinding,verbs=get;list;create;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the DSMaster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *DSMasterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	masterLogger.Info("dmMaster start reconcile logic")
	defer masterLogger.Info("dmMaster Reconcile end ---------------------------------------------")

	cluster := &dsv1alpha1.DSMaster{}

	if err := r.Client.Get(ctx, req.NamespacedName, cluster); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	desired := cluster.DeepCopy()

	sa := &corev1.ServiceAccount{}
	saReq := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: req.Namespace,
			Name:      dsv1alpha1.DsServiceAccount,
		},
	}
	err := r.Get(ctx, saReq.NamespacedName, sa)
	if apierrors.IsNotFound(err) {
		err := r.createServiceAccountIfNotExists(ctx, cluster)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

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
	// Until dsmaster-cluster is un-paused, we will reconcile to the  state of that point.
	if cluster.Spec.Paused {
		masterLogger.Info("ds-master control has been paused: ", "ds-master-name", cluster.Name)
		desired.Status.ControlPaused = true
		if err := r.Status().Patch(ctx, desired, client.MergeFrom(cluster)); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			} else {
				masterLogger.Error(err, "unexpected error when master update status in paused")
				return ctrl.Result{}, err
			}
		}
		r.recorder.Event(cluster, corev1.EventTypeNormal, "the master spec status is paused", "do nothing")
		return ctrl.Result{}, nil
	}

	// 1. First time we see the ds-master-cluster, initialize it
	if cluster.Status.Phase == dsv1alpha1.DsPhaseNone {
		desired.Status.Phase = dsv1alpha1.DsPhaseCreating
		masterLogger.Info("phase had been changed from  none ---> creating")
		if err := r.Client.Status().Patch(ctx, desired, client.MergeFrom(cluster)); err != nil {

			if apierrors.IsConflict(err) {
				return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, err
			} else {
				masterLogger.Error(err, "unexpected error when master update status in creating")
				return ctrl.Result{}, err
			}
		}
	}

	//2 ensure the headless service
	masterLogger.Info("Ensuring cluster service")

	if err := r.ensureMasterService(ctx, cluster); err != nil {
		return ctrl.Result{}, err
	}

	//2 ensure the headless service
	masterLogger.Info("Ensuring worker hpa")

	if err := r.ensureHPA(ctx, cluster); err != nil {
		return ctrl.Result{}, err
	}

	// 4. Ensure bootstrapped, we will block here util cluster is up and healthy
	masterLogger.Info("Ensuring cluster members")
	if requeue, err := r.ensureMembers(ctx, cluster); requeue {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	// 5. Ensure cluster scaled
	masterLogger.Info("Ensuring cluster scaled")
	if requeue, err := r.ensureScaled(ctx, cluster); requeue {
		return ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, err
	}

	// 6. Ensure cluster upgraded
	masterLogger.Info("Ensuring cluster upgraded")
	if requeue, err := r.ensureUpgraded(ctx, cluster); requeue {
		return ctrl.Result{Requeue: true}, err
	}

	desired.Status.Phase = dsv1alpha1.DsPhaseFinished
	if err := r.Status().Patch(ctx, desired, client.MergeFrom(cluster)); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		} else {
			masterLogger.Error(err, "unexpected error when master update status in finished")
			return ctrl.Result{}, err
		}
	}

	masterLogger.Info("******************************************************")
	desired.Status.Phase = dsv1alpha1.DsPhaseNone
	if err := r.Update(ctx, desired); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: false}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DSMasterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("master-controller")

	filter := &Predicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&dsv1alpha1.DSMaster{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Owns(&v2beta2.HorizontalPodAutoscaler{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&v1.Role{}).
		Owns(&v1.RoleBinding{}).
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
			r.recorder.Event(cluster, corev1.EventTypeWarning, "cannot create the new ds-master pod", "the ds-master pod had been created failed")
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
	masterLogger.Info("Starting add new member to cluster", "cluster", cluster.Name)
	defer masterLogger.Info("End add new member to cluster", "cluster", cluster.Name)

	// New Pod
	pod, err := r.newDSMasterPod(cluster)
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
	masterLogger.Info("begin delete pod", "pod name", pod.Name)
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

	masterLogger.Info("cluster.Spec.Version", "cluster.Spec.Version", cluster.Spec.Version)
	for _, memset := range ms {
		if r.predicateUpdate(memset, cluster) {
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
	podAppSelect, err := labels.NewRequirement(dsv1alpha1.DsAppName, selection.Equals, []string{dsv1alpha1.DsMasterLabel})
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
	namespacedName := types.NamespacedName{Namespace: cluster.Namespace, Name: dsv1alpha1.DsHeadLessServiceLabel}
	if err := r.Client.Get(ctx, namespacedName, service); err != nil {
		// Local cache not found
		if apierrors.IsNotFound(err) && !apierrors.IsAlreadyExists(err) {
			service = createMasterService(cluster)
			if err := controllerutil.SetControllerReference(cluster, service, r.Scheme); err != nil {
				return err
			}
			// Remote may already exist, so we will return err, for the next time, this code will not execute
			if err := r.Client.Create(ctx, service); err != nil {
				return err
			}

			r.recorder.Event(cluster, corev1.EventTypeNormal, "ds-operator-service created", "master headless service had been created")
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

		// Only care about size, repo,version and paused fields
		if oldC.Spec.Replicas != newC.Spec.Replicas {
			return true
		}

		if oldC.Spec.Paused != newC.Spec.Paused {
			return true
		}

		if oldC.Spec.Version != newC.Spec.Version {
			return true
		}

		if oldC.Spec.Repository != newC.Spec.Repository {
			return true
		}

		// If cluster has been marked as deleted, check if we have removed our finalizer
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

func (r *DSMasterReconciler) predicateUpdate(member *Member, cluster *dsv1alpha1.DSMaster) bool {
	return member.Version != cluster.Spec.Version
}

func (r *DSMasterReconciler) ensureHPA(ctx context.Context, cluster *dsv1alpha1.DSMaster) error {
	hpa := &v2beta2.HorizontalPodAutoscaler{}
	namespacedName := types.NamespacedName{Namespace: cluster.Namespace, Name: dsv1alpha1.DsWorkerHpa}
	if err := r.Client.Get(ctx, namespacedName, hpa); err != nil {
		// Local cache not found
		if apierrors.IsNotFound(err) && cluster.Spec.HpaPolicy != nil {
			hpa := r.createHPA(cluster)
			if err := controllerutil.SetControllerReference(cluster, hpa, r.Scheme); err != nil {
				masterLogger.Info("set controller worker hpa error")
				return err
			}
			// Remote may already exist, so we will return err, for the next time, this code will not execute
			if err := r.Client.Create(ctx, hpa); err != nil {
				masterLogger.Info("create worker hpa error")
				return err
			}
		}
	}

	if hpa.Kind != "" && cluster.Spec.HpaPolicy == nil {
		if err := r.deleteHPA(ctx, hpa); err != nil {
			masterLogger.Info("delete hpa error")
			return err
		}
	}
	return nil
}

// 创建 ServiceAccount
func (r *DSMasterReconciler) createServiceAccountIfNotExists(ctx context.Context, cluster *dsv1alpha1.DSMaster) (err error) {

	masterLogger.Info("start create service account.")

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsv1alpha1.DsServiceAccount,
			Namespace: cluster.Namespace,
		},
	}

	err = r.Create(ctx, sa)

	if err != nil {
		masterLogger.Error(err, "create service account error")
		return err
	}
	// binding the sa
	err = controllerutil.SetControllerReference(cluster, sa, r.Scheme)
	if err != nil {
		masterLogger.Error(err, "sa SetControllerReference error")
		return err
	}

	ro := &v1.Role{}
	namespacedName := types.NamespacedName{Namespace: cluster.Namespace, Name: dsv1alpha1.DsRole}
	if err := r.Client.Get(ctx, namespacedName, ro); err != nil {
		if apierrors.IsNotFound(err) && !apierrors.IsAlreadyExists(err) {
			// Remote may already exist, so we will return err, for the next time, this code will not execute
			ro := r.createRole(cluster)
			if err := controllerutil.SetControllerReference(cluster, ro, r.Scheme); err != nil {
				masterLogger.Info("set controller role  error")
				return err
			}
			masterLogger.Info("set  role  begin")
			if err := r.Client.Create(ctx, ro); err != nil {
				return err
			}
		}
	}

	rb := &v1.RoleBinding{}
	rbNamespacedName := types.NamespacedName{Namespace: cluster.Namespace, Name: dsv1alpha1.DsRoleBinding}
	if err := r.Client.Get(ctx, rbNamespacedName, rb); err != nil {
		if apierrors.IsNotFound(err) && !apierrors.IsAlreadyExists(err) {
			rb := r.createRoleBinding(cluster)
			if err := controllerutil.SetControllerReference(cluster, rb, r.Scheme); err != nil {
				masterLogger.Info("set controller  rolebinding error")
				return err
			}

			masterLogger.Info("set  rolebinding  begin")
			if err := r.Client.Create(ctx, rb); err != nil {
				return err
			}
		}
	}
	return nil
}
