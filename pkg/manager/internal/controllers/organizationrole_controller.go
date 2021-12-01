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
	"reflect"

	"github.com/go-logr/logr"
	"k8c.io/utils/pkg/util"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	corev1alpha1 "k8c.io/bulward/pkg/apis/core/v1alpha1"
	"k8c.io/bulward/pkg/utils/intersect"
)

type OrganizationRoleReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=bulward.io,resources=organizationroles,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=bulward.io,resources=organizationroles/status,verbs=get;update;patch

func (r *OrganizationRoleReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("organizationRole", req.NamespacedName)

	organizationRole := &corev1alpha1.OrganizationRole{}
	if err := r.Get(ctx, req.NamespacedName, organizationRole); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
	}

	if !organizationRole.GetDeletionTimestamp().IsZero() {
		if err := r.handleDeletion(ctx, log, organizationRole); err != nil {
			return ctrl.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return ctrl.Result{}, nil
	}

	if util.AddFinalizer(organizationRole, metav1.FinalizerDeleteDependents) {
		if err := r.Client.Update(ctx, organizationRole); err != nil {
			return ctrl.Result{}, fmt.Errorf("updating OrganizationRole finalizers: %w", err)
		}
	}

	acceptedRules, err := r.reconcileRole(ctx, organizationRole)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling Role: %w", err)
	}

	var isChanged bool
	if !reflect.DeepEqual(organizationRole.Status.AcceptedRules, acceptedRules) {
		organizationRole.Status.AcceptedRules = acceptedRules
		isChanged = true
	}
	if !organizationRole.IsReady() {
		// Update OrganizationRole Status
		organizationRole.Status.ObservedGeneration = organizationRole.Generation
		organizationRole.Status.SetCondition(corev1alpha1.OrganizationRoleCondition{
			Type:    corev1alpha1.OrganizationRoleReady,
			Status:  corev1alpha1.ConditionTrue,
			Reason:  "SetupComplete",
			Message: "OrganizationRole setup is complete.",
		})
		isChanged = true
	}
	if isChanged {
		if err := r.Status().Update(ctx, organizationRole); err != nil {
			return ctrl.Result{}, fmt.Errorf("updating OrganizationRole status: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *OrganizationRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	enqueueAllRoles := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(mapObject handler.MapObject) (out []ctrl.Request) {
			organizationRoles := &corev1alpha1.OrganizationRoleList{}
			if err := r.Client.List(context.Background(), organizationRoles); err != nil {
				// This will makes the manager crashes, and it will restart and reconcile all objects again.
				panic(fmt.Errorf("listting OrganizationRoles: %w", err))
			}
			for _, organizationRole := range organizationRoles.Items {
				out = append(out, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      organizationRole.Name,
						Namespace: organizationRole.Namespace,
					},
				})
			}
			return
		}),
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.OrganizationRole{}).
		Owns(&rbacv1.Role{}).
		Watches(&source.Kind{Type: &corev1alpha1.OrganizationRoleTemplate{}}, enqueueAllRoles).
		Complete(r)
}

func (r *OrganizationRoleReconciler) handleDeletion(ctx context.Context, log logr.Logger, organizationRole *corev1alpha1.OrganizationRole) error {
	// Update the OrganizationRole Status to Terminating.
	readyCondition, _ := organizationRole.Status.GetCondition(corev1alpha1.OrganizationRoleReady)
	if readyCondition.Status != corev1alpha1.ConditionFalse ||
		readyCondition.Status == corev1alpha1.ConditionFalse && readyCondition.Reason != corev1alpha1.OrganizationRoleTerminatingReason {
		organizationRole.Status.ObservedGeneration = organizationRole.Generation
		organizationRole.Status.SetCondition(corev1alpha1.OrganizationRoleCondition{
			Type:    corev1alpha1.OrganizationRoleReady,
			Status:  corev1alpha1.ConditionFalse,
			Reason:  corev1alpha1.OrganizationRoleTerminatingReason,
			Message: "OrganizationRole is being terminated",
		})
		if err := r.Status().Update(ctx, organizationRole); err != nil {
			return fmt.Errorf("updating OrganizationRole status: %w", err)
		}
	}
	return nil
}

func (r *OrganizationRoleReconciler) reconcileRole(ctx context.Context, organizationRole *corev1alpha1.OrganizationRole) (acceptedRules []rbacv1.PolicyRule, err error) {
	organizationRoleTemplates := &corev1alpha1.OrganizationRoleTemplateList{}
	if err = r.List(ctx, organizationRoleTemplates); err != nil {
		return
	}
	var maxRules []rbacv1.PolicyRule
	for _, template := range organizationRoleTemplates.Items {
		maxRules = append(maxRules, template.Spec.Rules...)
	}
	policyRoles := intersect.PolicyRules(maxRules, organizationRole.Spec.Rules)

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      organizationRole.Name,
			Namespace: organizationRole.Namespace,
		},
		Rules: policyRoles,
	}
	desiredRole := role.DeepCopy()
	if _, err = controllerutil.CreateOrUpdate(ctx, r.Client, role, func() error {
		if err = controllerutil.SetControllerReference(organizationRole, role, r.Scheme); err != nil {
			return fmt.Errorf("setting owner reference: %w", err)
		}
		role.Rules = desiredRole.Rules
		return nil
	}); err != nil {
		return nil, err
	}
	return role.Rules, nil
}
