/*
Copyright 2022 imliuda.

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
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	dsLogVolumeName       = "ds-log"
	dsLogVolumeMountDir   = "/opt/dolphinscheduler/logs"
	dsLogPVName           = "pv"
	dsShareVolumeName     = "ds-soft"
	dsShareVolumeMountDir = "/opt/soft"
)

func GetDsVersion(pod *corev1.Pod) string {
	return pod.Labels[dsv1alpha1.DsVersionLabel]
}

func SetDSVersion(pod *corev1.Pod, version string) {
	pod.Labels[dsv1alpha1.DsVersionLabel] = version
}

func GetPodNames(pods []*corev1.Pod) []string {
	if len(pods) == 0 {
		return nil
	}
	res := make([]string, 0)
	for _, p := range pods {
		res = append(res, p.Name)
	}
	return res
}

func applyPodPolicy(pod *corev1.Pod, policy *dsv1alpha1.PodPolicy) {
	if policy == nil {
		return
	}

	if policy.Affinity != nil {
		pod.Spec.Affinity = policy.Affinity
	}

	if len(policy.NodeSelector) != 0 {
		pod = PodWithNodeSelector(pod, policy.NodeSelector)
	}

	if len(policy.Tolerations) != 0 {
		pod.Spec.Tolerations = policy.Tolerations
	}

	mergeLabels(pod.Labels, policy.Labels)

	if &policy.Resources != nil {
		pod.Spec.Containers[0] = containerWithRequirements(pod.Spec.Containers[0], policy.Resources)
	}

	if len(policy.Envs) != 0 {
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, policy.Envs...)
	}

	for key, value := range policy.Annotations {
		pod.ObjectMeta.Annotations[key] = value
	}
}

func containerWithRequirements(c corev1.Container, r corev1.ResourceRequirements) corev1.Container {
	c.Resources = r
	return c
}

// PVCNameFromMember the way we get PVC name from the member name
func PVCNameFromMember(memberName string) string {
	return memberName
}

func ImageName(repo, version string) string {
	return fmt.Sprintf("%s:%v", repo, version)
}

func PodWithNodeSelector(p *corev1.Pod, ns map[string]string) *corev1.Pod {
	p.Spec.NodeSelector = ns
	return p
}

func LabelsForCluster(lbs string) map[string]string {
	return labels.Set{dsv1alpha1.DsAppName: lbs}
}

func LabelsForPV() map[string]string {
	return labels.Set{dsv1alpha1.DsAppName: dsLogPVName}
}

func LabelsForService() map[string]string {
	return labels.Set{dsv1alpha1.DsServiceLabel: dsv1alpha1.DsServiceLabelValue}
}

// AddDSVolumeToPod abstract the process of appending volume spec to pod spec
func AddDSVolumeToPod(pod *corev1.Pod, pvc *corev1.PersistentVolumeClaim) {
	vol := corev1.Volume{Name: dsLogVolumeName}
	if pvc != nil {
		vol.VolumeSource = corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvc.Name},
		}
	} else {
		vol.VolumeSource = corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, vol)
}

// AddLogVolumeToPod abstract the process of appending volume spec to pod spec
func AddLogVolumeToPod(pod *corev1.Pod, pvcName string) {
	vol := corev1.Volume{Name: dsLogVolumeName}

	vom := corev1.VolumeMount{
		Name:      dsLogVolumeName,
		MountPath: dsLogVolumeMountDir,
		SubPath:   pod.Name,
	}

	if len(pvcName) != 0 {
		vol.VolumeSource = corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName},
		}
	} else {
		vol.VolumeSource = corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, vol)

	pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, vom)
}

// AddLibVolumeToPod abstract the process of appending volume /opt/soft spec to pod spec,it is shared by all worker nodes,and it is read only
// Suggest to mount a share volume in production env directly
func AddLibVolumeToPod(pod *corev1.Pod, pvcName string) {
	vol := corev1.Volume{Name: dsShareVolumeName}

	vom := corev1.VolumeMount{
		Name:      dsShareVolumeName,
		MountPath: dsShareVolumeMountDir,
		ReadOnly:  true,
	}

	if len(pvcName) != 0 {
		vol.VolumeSource = corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName},
		}
	} else {
		vol.VolumeSource = corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, vol)

	pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, vom)
}

// NewLogPVC create PVC  from dsMaster pod's PVC spec
//func NewLogPVC(cluster *dsv1alpha1.DSMaster, pod *corev1.Pod, storageClassName string) *corev1.PersistentVolumeClaim {
//
//    pvc := &corev1.PersistentVolumeClaim{
//        ObjectMeta: metav1.ObjectMeta{
//            GenerateName: cluster.Name + "-pvc",
//            Namespace:    cluster.Namespace,
//            Labels:       LabelsForCluster(dsLogVolumeName),
//        },
//        Spec: corev1.PersistentVolumeClaimSpec{
//            AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
//            Resources: corev1.ResourceRequirements{
//                Requests: corev1.ResourceList{
//                    corev1.ResourceStorage: resource.MustParse(cluster.Spec.LogCapacity),
//                },
//            },
//            StorageClassName: &storageClassName,
//            Selector: &metav1.LabelSelector{
//                MatchLabels: LabelsForPV(),
//            },
//        },
//    }
//    return pvc
//}

// NewDSWorkerPodPVC create PVC object from dsMaster pod's PVC spec
func NewDSWorkerPodPVC(cluster *dsv1alpha1.DSWorker, pod *corev1.Pod, lbs string) *corev1.PersistentVolumeClaim {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PVCNameFromMember(pod.Name),
			Namespace: cluster.Namespace,
			Labels:    LabelsForCluster(lbs),
		},
		Spec: *cluster.Spec.Pod.PersistentVolumeClaimSpec,
	}
	return pvc
}

// mergeLabels merges l2 into l1. Conflicting label will be skipped.
func mergeLabels(l1, l2 map[string]string) {
	for k, v := range l2 {
		if _, ok := l1[k]; ok {
			continue
		}
		l1[k] = v
	}
}

func IsRunningAndReady(pod *corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodRunning && IsPodReady(pod)
}

// IsPodReady returns true if a pod is ready; false otherwise.
func IsPodReady(pod *corev1.Pod) bool {
	return IsPodReadyConditionTrue(pod.Status)
}

// IsPodReadyConditionTrue returns true if a pod is ready; false otherwise.
func IsPodReadyConditionTrue(status corev1.PodStatus) bool {
	condition := GetPodReadyCondition(status)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// GetPodReadyCondition extracts the pod ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetPodReadyCondition(status corev1.PodStatus) *corev1.PodCondition {
	_, condition := GetPodCondition(&status, corev1.PodReady)
	return condition
}

// GetPodCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func GetPodCondition(status *corev1.PodStatus, conditionType corev1.PodConditionType) (int, *corev1.PodCondition) {
	if status == nil {
		return -1, nil
	}
	return GetPodConditionFromList(status.Conditions, conditionType)
}

// GetPodConditionFromList extracts the provided condition from the given list of condition and
// returns the index of the condition and the condition. Returns -1 and nil if the condition is not present.
func GetPodConditionFromList(conditions []corev1.PodCondition, conditionType corev1.PodConditionType) (int, *corev1.PodCondition) {
	if conditions == nil {
		return -1, nil
	}
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return i, &conditions[i]
		}
	}
	return -1, nil
}
