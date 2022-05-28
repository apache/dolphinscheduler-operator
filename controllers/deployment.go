package controllers

import (
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// IsDeploymentAvailable returns true if a pod is ready; false otherwise.
func IsDeploymentAvailable(deployment *v1.Deployment) bool {
	return IsDeploymentAvailableConditionTrue(deployment.Status)
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
