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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DSWorkerSpec defines the desired state of DSWorker
type DSWorkerSpec struct {
	//Datasource is the config of database
	Datasource *DateSourceTemplate `json:"datasource"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Version is the expected version of the ds cluster.
	// The ds-operator will eventually make the ds cluster version
	// equal to the expected version.
	// If version is not set, default is "3.0.0-alpha".
	// +kubebuilder:default="3.0.0-alpha"
	Version string `json:"version,omitempty"`

	// Repository is the name of the repository that hosts
	// ds container images. It should be direct clone of the repository in official
	// By default, it is `apache/dolphinscheduler-worker`.
	// +kubebuilder:default=apache/dolphinscheduler-worker
	Repository string `json:"repository,omitempty"`

	// Replicas is the expected size of the ms-worker.
	// The ds-worker-operator will eventually make the size of the running
	//  equal to the expected size.
	// The vaild range of the size is from 1 to 7.
	// +kubebuilder:default=3
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=7
	Replicas int `json:"replicas"`

	//ZookeeperConnect  is the address string of zookeeper ,and it will be written to ENV
	ZookeeperConnect string `json:"zookeeper_connect,omitempty"`

	// Pod defines the policy to create pod for the dm-worker pod.
	// Updating Pod does not take effect on any existing dm-worker pods.
	Pod *PodPolicy `json:"pod,omitempty"`

	// Paused is to pause the control of the operator for the ds-worker .
	// +kubebuilder:default=false
	Paused bool `json:"paused,omitempty"`

	//LogPvcName defines the address of log pvc ,the position is /opt/dolphinscheduler/logs
	LogPvcName string `json:"log_pvc_name,omitempty"`

	//LibPvcName define the address of lib pvc,the position is /opt/soft
	LibPvcName string `json:"lib_pvc_name,omitempty"`

	//AlertConfig is the config of alertService
	AlertConfig *AlertConfig `json:"alert_config,omitempty"`
}

// DSWorkerStatus defines the observed state of DSWorker
type DSWorkerStatus struct {
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

	// Members are the dsWorker members in the cluster
	Members MembersStatus `json:"members,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DSWorker is the Schema for the dsworkers API
type DSWorker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DSWorkerSpec   `json:"spec,omitempty"`
	Status DSWorkerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DSWorkerList contains a list of DSWorker
type DSWorkerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DSWorker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DSWorker{}, &DSWorkerList{})
}

type AlertConfig struct {
	ServiceUrl string `json:"service_url,omitempty"`
	Port       string `json:"port,omitempty"`
}
