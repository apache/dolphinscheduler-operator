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
	dsv1alpha1 "dolphinscheduler-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func createApiService(cluster *dsv1alpha1.DSApi) *corev1.Service {
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsv1alpha1.DsApiServiceValue,
			Namespace: cluster.Namespace,
			Labels:    map[string]string{dsv1alpha1.DsAppName: dsv1alpha1.DsApiServiceValue},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeNodePort,
			Selector: map[string]string{dsv1alpha1.DsAppName: dsv1alpha1.DsApi},
			Ports: []corev1.ServicePort{
				{
					Protocol: corev1.ProtocolTCP,
					Port:     *int32Ptr(int32(dsv1alpha1.DsApiPort)),
					TargetPort: intstr.IntOrString{
						IntVal: dsv1alpha1.DsApiPort,
					},
					NodePort: cluster.Spec.NodePort,
				},
			},
		},
	}
	return &service
}

func createApiDeployment(cluster *dsv1alpha1.DSApi) *v1.Deployment {
	ApiDeployment := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsv1alpha1.DsApiDeploymentValue,
			Namespace: cluster.Namespace,
		},
		Spec: v1.DeploymentSpec{
			Replicas: int32Ptr(cluster.Spec.Replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					dsv1alpha1.DsAppName: dsv1alpha1.DsApi,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						dsv1alpha1.DsAppName: dsv1alpha1.DsApi,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            dsv1alpha1.DsApi,
						Image:           ImageName(cluster.Spec.Repository, cluster.Spec.Version),
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env: []corev1.EnvVar{
							{
								Name:  dsv1alpha1.EnvZookeeper,
								Value: cluster.Spec.ZookeeperConnect,
							},
							{
								Name:  dsv1alpha1.DataSourceDriveName,
								Value: cluster.Spec.Datasource.DriveName,
							},
							{
								Name:  dsv1alpha1.DataSourceUrl,
								Value: cluster.Spec.Datasource.Url,
							},
							{
								Name:  dsv1alpha1.DataSourceUserName,
								Value: cluster.Spec.Datasource.UserName,
							},
							{
								Name:  dsv1alpha1.DataSourcePassWord,
								Value: cluster.Spec.Datasource.Password,
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: dsv1alpha1.DsApiPort,
						},
						},
					},
					},
				},
			},
		},
	}
	return &ApiDeployment
}
