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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DSApiSpec defines the desired state of DSApi
type DSApiSpec struct {
	Datasource *DateSourceTemplate `json:"datasource"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Version is the expected version of the ds cluster.
	// The ds-operator will eventually make the ds cluster version
	// equal to the expected version.
	// If version is not set, default is "3.0.0-alpha".
	// +kubebuilder:default="3.0.0-alpha"
	Version string `json:"version,omitempty"`

	ZookeeperConnect string `json:"zookeeper_connect,omitempty"`

	// Repository is the name of the repository that hosts
	// ds container images. It should be direct clone of the repository in official
	// By default, it is `apache/dolphinscheduler-master`.
	// +kubebuilder:default=apache/dolphinscheduler-master
	Repository string `json:"repository,omitempty"`

	// Replicas is the expected size of the ms-master.
	// The ds-master-operator will eventually make the size of the running
	//  equal to the expected size.
	// The vaild range of the size is from 1 to 7.
	// +kubebuilder:default=3
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=7
	Replicas int32 `json:"replicas"`

	// Pod defines the policy to create pod for the dm-master pod.
	// Pod defines the policy to create pod for the dm-master pod.
	Deployment *DeploymentPolicy `json:"deployment,omitempty"`

	// Paused is to pause the control of the operator for the ds-master .
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`

	//LogPvcName defines the  log capacity of application ,the position is /opt/dolphinscheduler/logs eg 20Gi
	LogPvcName string `json:"log_pvc_name,omitempty"`

	//NodePort is the port node exposed if no nodePort it will be lb type
	NodePort int32 `json:"node_port,omitempty"`
}

// DSApiStatus defines the observed state of DSApi
type DSApiStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Phase is the cluster running phase
	// +kubebuilder:validation:Enum="";Creating;Running;Failed;Finished
	Phase DsPhase `json:"phase,omitempty"`

	// ControlPaused indicates the operator pauses the control of the cluster.
	// +kubebuilder:default=false
	ControlPaused bool `json:"controlPaused,omitempty"`

	// Condition keeps track of all cluster conditions, if they exist.
	Conditions []DsCondition `json:"conditions,omitempty"`

	// Replicas is the current size of the cluster
	// +kubebuilder:default=0
	Replicas int `json:"replicas,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DSApi is the Schema for the dsapis API
type DSApi struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DSApiSpec   `json:"spec,omitempty"`
	Status DSApiStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DSApiList contains a list of DSApi
type DSApiList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DSApi `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DSApi{}, &DSApiList{})
}
