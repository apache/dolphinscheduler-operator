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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var dsmasterlog = logf.Log.WithName("dsmaster-resource")

func (r *DSMaster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-ds-apache-dolphinscheduler-dev-v1alpha1-dsmaster,mutating=true,failurePolicy=fail,sideEffects=None,groups=ds.apache.dolphinscheduler.dev,resources=dsmasters,verbs=create;update,versions=v1alpha1,name=mdsmaster.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &DSMaster{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *DSMaster) Default() {

	if r.Spec.HpaPolicy != nil {
		if &r.Spec.HpaPolicy.MinReplicas == nil {
			r.Spec.HpaPolicy.MinReplicas = int32(1)
		}

		if &r.Spec.HpaPolicy.MaxReplicas == nil {
			r.Spec.HpaPolicy.MinReplicas = int32(5)
		}
	}

}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-ds-apache-dolphinscheduler-dev-v1alpha1-dsmaster,mutating=false,failurePolicy=fail,sideEffects=None,groups=ds.apache.dolphinscheduler.dev,resources=dsmasters,verbs=create;update,versions=v1alpha1,name=vdsmaster.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &DSMaster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *DSMaster) ValidateCreate() error {
	var allErrs field.ErrorList

	if r.Spec.HpaPolicy != nil {
		if &r.Spec.HpaPolicy.MinReplicas != nil && &r.Spec.HpaPolicy.MaxReplicas != nil && r.Spec.HpaPolicy.MinReplicas > r.Spec.HpaPolicy.MaxReplicas {
			return apierrors.NewInvalid(
				schema.GroupKind{Group: "ds", Kind: "DSMaster"},
				r.Name,
				allErrs)
		}

	} else {
		dsmasterlog.Info("Hpa `s replicas is valid")
		return nil
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *DSMaster) ValidateUpdate(old runtime.Object) error {
	dsmasterlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *DSMaster) ValidateDelete() error {
	dsmasterlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
