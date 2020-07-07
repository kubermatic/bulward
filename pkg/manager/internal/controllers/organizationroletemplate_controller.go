/*
Copyright 2020 The Bulward Authors.

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
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
)

// OrganizationRoleTemplateReconciler reconciles a Organization object
type OrganizationRoleTemplateReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=bulward.io,resources=organizationroletemplates,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=bulward.io,resources=organizationroletemplates/status,verbs=get;update;patch

func (r *OrganizationRoleTemplateReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

	organizationRoleTemplate := &corev1alpha1.OrganizationRoleTemplate{}
	if err := r.Get(ctx, req.NamespacedName, organizationRoleTemplate); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !organizationRoleTemplate.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, organizationRoleTemplate); err != nil {
			return ctrl.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return ctrl.Result{}, nil
	}

	if !organizationRoleTemplate.IsReady() {
		// Update OrganizationRoleTemplate Status
		organizationRoleTemplate.Status.ObservedGeneration = organizationRoleTemplate.Generation
		organizationRoleTemplate.Status.SetCondition(corev1alpha1.OrganizationRoleTemplateCondition{
			Type:    corev1alpha1.OrganizationRoleTemplateReady,
			Status:  corev1alpha1.ConditionTrue,
			Reason:  "SetupComplete",
			Message: "OrganizationRoleTemplate setup is complete.",
		})
		if err := r.Status().Update(ctx, organizationRoleTemplate); err != nil {
			return ctrl.Result{}, fmt.Errorf("updating OrganizationRoleTemplate status: %w", err)
		}
	}
	return ctrl.Result{}, nil
}

func (r *OrganizationRoleTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.OrganizationRoleTemplate{}).
		Complete(r)
}

// handleDeletion handles the deletion of the OrganizationRoleTemplate object:
func (r *OrganizationRoleTemplateReconciler) handleDeletion(ctx context.Context, organizationRoleTemplate *corev1alpha1.OrganizationRoleTemplate) error {
	// Update the OrganizationRoleTemplate Status to Terminating.
	readyCondition, _ := organizationRoleTemplate.Status.GetCondition(corev1alpha1.OrganizationRoleTemplateReady)
	if readyCondition.Status != corev1alpha1.ConditionFalse ||
		readyCondition.Status == corev1alpha1.ConditionFalse && readyCondition.Reason != corev1alpha1.OrganizationRoleTemplateTerminatingReason {
		organizationRoleTemplate.Status.ObservedGeneration = organizationRoleTemplate.Generation
		organizationRoleTemplate.Status.SetCondition(corev1alpha1.OrganizationRoleTemplateCondition{
			Type:    corev1alpha1.OrganizationRoleTemplateReady,
			Status:  corev1alpha1.ConditionFalse,
			Reason:  corev1alpha1.OrganizationRoleTemplateTerminatingReason,
			Message: "OrganizationRoleTemplate is being terminated",
		})
		if err := r.Status().Update(ctx, organizationRoleTemplate); err != nil {
			return fmt.Errorf("updating OrganizationRoleTemplate status: %w", err)
		}
	}

	return nil
}
