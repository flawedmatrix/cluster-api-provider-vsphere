/*
Copyright 2021 The Kubernetes Authors.

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

package v1beta1

import (
	"context"
	"fmt"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/cluster-api/util/topology"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const machineTemplateImmutableMsg = "VSphereMachineTemplate spec.template.spec field is immutable. Please create a new resource instead."

func (v *VSphereMachineTemplateWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&VSphereMachineTemplate{}).
		WithValidator(v).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-vspheremachinetemplate,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=vspheremachinetemplates,versions=v1beta1,name=validation.vspheremachinetemplate.infrastructure.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1

// VSphereMachineTemplateWebhook implements a custom validation webhook for DockerMachineTemplate.
// +kubebuilder:object:generate=false
type VSphereMachineTemplateWebhook struct{}

var _ webhook.CustomValidator = &VSphereMachineTemplateWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (v *VSphereMachineTemplateWebhook) ValidateCreate(_ context.Context, raw runtime.Object) error {
	obj, ok := raw.(*VSphereMachineTemplate)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a VSphereMachineTemplate but got a %T", raw))
	}

	var allErrs field.ErrorList
	spec := obj.Spec.Template.Spec

	if spec.Network.PreferredAPIServerCIDR != "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "PreferredAPIServerCIDR"), spec.Network.PreferredAPIServerCIDR, "cannot be set, as it will be removed and is no longer used"))
	}
	if spec.ProviderID != nil {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "template", "spec", "providerID"), "cannot be set in templates"))
	}
	for _, device := range spec.Network.Devices {
		if len(device.IPAddrs) != 0 {
			allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "template", "spec", "network", "devices", "ipAddrs"), "cannot be set in templates"))
		}
	}
	return aggregateObjErrors(obj.GroupVersionKind().GroupKind(), obj.Name, allErrs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (v *VSphereMachineTemplateWebhook) ValidateUpdate(ctx context.Context, oldRaw runtime.Object, newRaw runtime.Object) error {
	newObj, ok := newRaw.(*VSphereMachineTemplate)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a VSphereMachineTemplate but got a %T", newRaw))
	}
	oldObj, ok := oldRaw.(*VSphereMachineTemplate)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a VSphereMachineTemplate but got a %T", oldRaw))
	}

	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a admission.Request inside context: %v", err))
	}

	var allErrs field.ErrorList
	if !topology.ShouldSkipImmutabilityChecks(req, newObj) &&
		!reflect.DeepEqual(newObj.Spec.Template.Spec, oldObj.Spec.Template.Spec) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "template", "spec"), newObj, machineTemplateImmutableMsg))
	}
	return aggregateObjErrors(newObj.GroupVersionKind().GroupKind(), newObj.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (v *VSphereMachineTemplateWebhook) ValidateDelete(_ context.Context, _ runtime.Object) error {
	return nil
}
