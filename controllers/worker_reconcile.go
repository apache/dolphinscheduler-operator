/*
Copyright 2022 nobolity.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *DSWorkerReconciler) podMemberSet(ctx context.Context, cluster *dsv1alpha1.DSWorker) (MemberSet, error) {
	members := MemberSet{}
	pods := &corev1.PodList{}

	if err := r.Client.List(ctx, pods, client.InNamespace(cluster.Namespace),
		client.MatchingLabels(LabelsForCluster(dsv1alpha1.DsWorkerLabel))); err != nil {
		return members, err
	}

	if len(pods.Items) > 0 {
		for _, pod := range pods.Items {
			if pod.ObjectMeta.DeletionTimestamp.IsZero() {
				m := &Member{
					Name:            pod.Name,
					Namespace:       pod.Namespace,
					Created:         true,
					Version:         pod.Labels[dsv1alpha1.DsVersionLabel],
					Phase:           string(pod.Status.Phase),
					RunningAndReady: IsRunningAndReady(&pod),
				}
				members.Add(m)
			}
		}
	}

	return members, nil
}

func newDSWorkerPod(cr *dsv1alpha1.DSWorker) *corev1.Pod {
	var podName = cr.Name + "-pod" + dsv1alpha1.RandStr(6)
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: cr.Namespace,
			Labels: map[string]string{dsv1alpha1.DsAppName: dsv1alpha1.DsWorkerLabel,
				dsv1alpha1.DsVersionLabel: ImageName(cr.Spec.Repository, cr.Spec.Version),
				dsv1alpha1.DsServiceLabel: dsv1alpha1.DsServiceLabelValue,
			},
		},
		Spec: corev1.PodSpec{
			Hostname:  podName,
			Subdomain: dsv1alpha1.DsServiceLabelValue,
			Containers: []corev1.Container{
				{
					Name:            cr.Name,
					Image:           ImageName(cr.Spec.Repository, cr.Spec.Version),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Env: []corev1.EnvVar{{
						Name:  dsv1alpha1.EnvZookeeper,
						Value: cr.Spec.ZookeeperConnect,
					}, {
						Name:  dsv1alpha1.DataSourceDriveName,
						Value: cr.Spec.Datasource.DriveName,
					},
						{
							Name:  dsv1alpha1.DataSourceUrl,
							Value: cr.Spec.Datasource.Url,
						},
						{
							Name:  dsv1alpha1.DataSourceUserName,
							Value: cr.Spec.Datasource.UserName,
						},
						{
							Name:  dsv1alpha1.DataSourcePassWord,
							Value: cr.Spec.Datasource.Password,
						},
					},
					Command: []string{
						"/bin/sh", "-c",
					},
					Args: []string{"sed -i 's/alert-listen-host: localhost/alert-listen-host: $(DS_ALERT_SERVICE_SERVICE_HOST)/g' conf/application.yaml ;" +
						" sed -i 's/50052/$(DS_ALERT_SERVICE_SERVICE_PORT)/g' conf/application.yaml ; " +
						"./bin/start.sh"},
				},
			},
		},
	}
}

func (r *DSWorkerReconciler) newDSWorkerPod(ctx context.Context, cluster *dsv1alpha1.DSWorker) (*corev1.Pod, error) {
	// Create pod
	pod := newDSWorkerPod(cluster)
	if err := controllerutil.SetControllerReference(cluster, pod, r.Scheme); err != nil {
		return nil, err
	}
	AddLogVolumeToPod(pod, cluster.Spec.LogPvcName)
	AddLibVolumeToPod(pod, cluster.Spec.LibPvcName)
	applyPodPolicy(pod, cluster.Spec.Pod)
	return pod, nil
}
