package controllers

import (
	dsv1alpha1 "dolphinscheduler-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// IsDeploymentAvailable returns true if a pod is ready; false otherwise.
func IsDeploymentAvailable(deployment *v1.Deployment) bool {
	return IsDeploymentAvailableConditionTrue(deployment.Status)
}

func applyDeploymentPolicy(deployment *v1.Deployment, policy *dsv1alpha1.DeploymentPolicy) {
	if policy == nil {
		return
	}

	if policy.Affinity != nil {
		deployment.Spec.Template.Spec.Affinity = policy.Affinity
	}

	if len(policy.Tolerations) != 0 {
		deployment.Spec.Template.Spec.Tolerations = policy.Tolerations
	}

	mergeLabels(deployment.Labels, policy.Labels)

	if &policy.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0] = containerWithRequirements(deployment.Spec.Template.Spec.Containers[0], policy.Resources)
	}

	if len(policy.Envs) != 0 {
		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, policy.Envs...)
	}

	for key, value := range policy.Annotations {
		deployment.ObjectMeta.Annotations[key] = value
	}
}

// IsDeploymentAvailableConditionTrue returns true if a deployment is available; false otherwise.
func IsDeploymentAvailableConditionTrue(status v1.DeploymentStatus) bool {
	condition := GetDeploymentAvailableCondition(status)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// GetDeploymentReadyCondition extracts the deployment available condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetDeploymentAvailableCondition(status v1.DeploymentStatus) *v1.DeploymentCondition {
	_, condition := GetDeploymentCondition(&status, v1.DeploymentAvailable)
	return condition
}

// GetDeploymentCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func GetDeploymentCondition(status *v1.DeploymentStatus, conditionType v1.DeploymentConditionType) (int, *v1.DeploymentCondition) {
	if status == nil {
		return -1, nil
	}
	return GetDeploymentConditionFromList(status.Conditions, conditionType)
}

// GetDeploymentConditionFromList extracts the provided condition from the given list of condition and
// returns the index of the condition and the condition. Returns -1 and nil if the condition is not present.
func GetDeploymentConditionFromList(conditions []v1.DeploymentCondition, conditionType v1.DeploymentConditionType) (int, *v1.DeploymentCondition) {
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
