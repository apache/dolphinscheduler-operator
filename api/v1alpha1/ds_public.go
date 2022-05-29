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

	ClusterMembersAnnotation      = "github.com/nobolity/members"
	ClusterUpgradeAnnotation      = "github.com/nobolity/upgrade"
	ClusterBootStrappedAnnotation = "github.com/nobolity/bootstrapped"

	DsAppName              = "app"
	DsVersionLabel         = "ds-version"
	FinalizerName          = "github.com.nobolity.dolphinscheduler-operator"
	EnvZookeeper           = "REGISTRY_ZOOKEEPER_CONNECT_STRING"
	DsServiceLabel         = "service-name"
	DsServiceLabelValue    = "ds-service"
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
	// dm-master cluster.
	// "app" and "dm-master_*" labels are reserved for the internal use of the dm-master operator.
	// Do not overwrite them.
	Labels map[string]string `json:"labels,omitempty"`

	// NodeSelector specifies a map of key-value pairs. For the pod to be eligible
	// to run on a node, the node must have each of the indicated key-value pairs as
	// labels.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// The scheduling constraints on dm-master pods.
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// **DEPRECATED**. Use Affinity instead.
	AntiAffinity bool `json:"antiAffinity,omitempty"`

	// Resources is the resource requirements for the dm-master container.
	// This field cannot be updated once the cluster is created.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Tolerations specifies the pod's tolerations.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// List of environment variables to set in the dm-master container.
	// This is used to configure dm-master process. dm-master cluster cannot be created, when
	// bad environement variables are provided. Do not overwrite any flags used to
	// bootstrap the cluster (for example `--initial-cluster` flag).
	// This field cannot be updated.
	Envs []corev1.EnvVar `json:"dm-masterEnv,omitempty"`

	// PersistentVolumeClaimSpec is the spec to describe PVC for the dm-master container
	// This field is optional. If no PVC spec, dm-master container will use emptyDir as volume
	// Note. This feature is in alpha stage. It is currently only used as non-stable storage,
	// not the stable storage. Future work need to make it used as stable storage.
	PersistentVolumeClaimSpec *corev1.PersistentVolumeClaimSpec `json:"persistentVolumeClaimSpec,omitempty"`

	// Annotations specifies the annotations to attach to pods the operator creates for the
	// dm-master cluster.
	// The "dm-master.version" annotation is reserved for the internal use of the dm-master operator.
	Annotations map[string]string `json:"annotations,omitempty"`

	// SecurityContext specifies the security context for the entire pod
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`
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
