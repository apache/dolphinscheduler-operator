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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"math/rand"
	"time"
)

type DsPhase string
type DsConditionType string

const (
	DsPhaseNone     DsPhase = ""
	DsPhaseCreating         = "Creating"
	DsPhaseRunning          = "Running"
	DsPhaseFailed           = "Failed"
	DsPhaseFinished         = "Finished"

	DsConditionAvailable DsConditionType = "Available"
	DsConditionScaling                   = "Scaling"
	DsConditionUpgrading                 = "Upgrading"

	DsAppName              = "app"
	APIVersion             = "ds.apache.dolphinscheduler.dev/v1alpha1"
	DsVersionLabel         = "ds-version"
	DsLogVolumeName        = "ds-log"
	DsLogVolumeMountDir    = "/opt/dolphinscheduler/logs"
	DsShareVolumeName      = "ds-soft"
	DsShareVolumeMountDir  = "/opt/soft"
	DSVersion              = "v1alpha1"
	FinalizerName          = "github.com.nobolity.dolphinscheduler-operator"
	EnvZookeeper           = "REGISTRY_ZOOKEEPER_CONNECT_STRING"
	DsServiceLabel         = "service-name"
	DsServiceLabelValue    = "ds-service"
	DsMasterLabel          = "ds-master"
	DsHeadLessServiceLabel = "ds-operator-service"
	DsWorkerLabel          = "ds-worker"
	DsWorkerKind           = "DSWorker"
	DsAlert                = "ds-alert"
	DsAlertServiceValue    = "ds-alert-service"
	DsAlertDeploymentValue = "ds-alert-deployment"
	DsApi                  = "ds-api"
	DsApiServiceValue      = "ds-api-service"
	DsApiDeploymentValue   = "ds-api-deployment"
	DataSourceDriveName    = "SPRING_DATASOURCE_DRIVER_CLASS_NAME"
	DataSourceUrl          = "SPRING_DATASOURCE_URL"
	DataSourceUserName     = "SPRING_DATASOURCE_USERNAME"
	DataSourcePassWord     = "SPRING_DATASOURCE_PASSWORD"
	DsApiPort              = 12345
	DsAlertPort            = 50052
	DsWorkerHpa            = "ds-worker-hpa"
)

// DsCondition represents one current condition of a ds cluster.
// A condition might not show up if it is not happening.
// For example, if a cluster is not upgrading, the Upgrading condition would not show up.
// If a cluster is upgrading and encountered a problem that prevents the upgrade,
// the Upgrading condition's status will would be False and communicate the problem back.
type DsCondition struct {
	// Type of cluster condition.
	Type DsConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime string `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// PodPolicy defines the policy to create pod for the dm-master container.
type PodPolicy struct {
	// Labels specifies the labels to attach to pods the operator creates for the
	// Do not overwrite them.
	Labels map[string]string `json:"labels,omitempty"`

	// NodeSelector specifies a map of key-value pairs. For the pod to be eligible
	// to run on a node, the node must have each of the indicated key-value pairs as
	// labels.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Resources is the resource requirements for the  container.
	// This field cannot be updated once the cluster is created.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Tolerations specifies the pod's tolerations.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// List of environment variables to set in the  container.
	// This is used to configure process. cluster cannot be created, when
	// bad environement variables are provided. Do not overwrite any flags used to
	// bootstrap the cluster (for example `--initial-cluster` flag).
	// This field cannot be updated.
	Envs []corev1.EnvVar `json:"envs,omitempty"`

	// Annotations specifies the annotations to attach to pods the operator creates for the cluster.
	Annotations map[string]string `json:"annotations,omitempty"`

	// SecurityContext specifies the security context for the entire pod
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`
}

type DeploymentPolicy struct {
	Labels      map[string]string           `json:"labels,omitempty"`
	Annotations map[string]string           `json:"annotations,omitempty"`
	Envs        []corev1.EnvVar             `json:"envs,omitempty"`
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
	Affinity    *corev1.Affinity            `json:"affinity,omitempty"`
	// Tolerations specifies the pod's tolerations.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

type HpaPolicy struct {
	MinReplicas           int32 `json:"min_replicas,omitempty"`
	MaxReplicas           int32 `json:"max_replicas,omitempty"`
	CPUAverageUtilization int32 `json:"cpu_average_utilization,omitempty"`
	MEMAverageUtilization int32 `json:"mem_average_utilization,omitempty"`
}

type MembersStatus struct {
	// Ready are the dsMaster members that are ready to serve requests
	// The member names are the same as the dsMaster pod names
	Ready []string `json:"ready,omitempty"`
	// Unready are the etcd members not ready to serve requests
	Unready []string `json:"unready,omitempty"`
}

func RandStr(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := []byte{}
	rand.Seed(time.Now().UnixNano() + int64(rand.Intn(100)))
	for i := 0; i < length; i++ {
		result = append(result, bytes[rand.Intn(len(bytes))])
	}
	return string(result)
}

type DateSourceTemplate struct {
	DriveName string `json:"drive_name"`
	Url       string `json:"url"`
	UserName  string `json:"username"`
	Password  string `json:"password"`
}
