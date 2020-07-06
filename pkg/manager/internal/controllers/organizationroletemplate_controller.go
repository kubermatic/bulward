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
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

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
// +kubebuilder:rbac:groups=bulward.io,resources=organizations,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=apiserver.bulward.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete;bind
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete

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

	organizations := &corev1alpha1.OrganizationList{}
	if err := r.Client.List(ctx, organizations); err != nil {
		return ctrl.Result{}, fmt.Errorf("listing Organizations: %w", err)
	}

	var targets []corev1alpha1.OrganizationRoleTemplateTarget
	for _, organization := range organizations.Items {
		if organization.Status.Namespace != nil && organization.Status.Namespace.Name != "" {
			if err := r.reconcileRBACForOrganization(ctx, organizationRoleTemplate, &organization); err != nil {
				return ctrl.Result{}, fmt.Errorf("reconcling Organization Role: %w", err)
			}
			targets = append(targets, corev1alpha1.OrganizationRoleTemplateTarget{
				Kind:               organization.Kind,
				APIGroup:           "bulward.io",
				Name:               organization.Name,
				ObservedGeneration: organization.Status.ObservedGeneration,
			})
		}
	}

	var isChanged bool
	if !reflect.DeepEqual(targets, organizationRoleTemplate.Status.Targets) {
		organizationRoleTemplate.Status.Targets = targets
		isChanged = true
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
		isChanged = true
	}

	if isChanged {
		if err := r.Status().Update(ctx, organizationRoleTemplate); err != nil {
			return ctrl.Result{}, fmt.Errorf("updating OrganizationRoleTemplate status: %w", err)
		}
	}
	return ctrl.Result{}, nil
}

func (r *OrganizationRoleTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.OrganizationRoleTemplate{}).
		Watches(&source.Kind{Type: &corev1alpha1.Organization{}}, &handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(mapObject handler.MapObject) (out []ctrl.Request) {
				organization := mapObject.Object.(*corev1alpha1.Organization)
				if !organization.IsReady() {
					return
				}
				templates := &corev1alpha1.OrganizationRoleTemplateList{}
				if err := r.Client.List(context.Background(), templates); err != nil {
					// This will makes the manager crashes, and it will restart and reconcile all objects again.
					panic(fmt.Errorf("listting OrganizationRoleTemplate: %w", err))
				}
				for _, template := range templates.Items {
					out = append(out, ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name: template.Name,
						},
					})
				}
				return
			}),
		}).
		Complete(r)
}

// handleDeletion handles the deletion of the OrganizationRoleTemplate object:
func (r *OrganizationRoleTemplateReconciler) handleDeletion(ctx context.Context, organizationRoleTemplate *corev1alpha1.OrganizationRoleTemplate) error {
	// Update the OrganizationRoleTemplate Status to Terminating.
	readyCondition, _ := organizationRoleTemplate.Status.GetCondition(corev1alpha1.OrganizationRoleTemplateReady)
	if readyCondition.Status != corev1alpha1.ConditionFalse ||
		readyCondition.Status == corev1alpha1.ConditionFalse && readyCondition.Reason != corev1alpha1.OrganizationTerminatingReason {
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

func (r *OrganizationRoleTemplateReconciler) reconcileRBAC(ctx context.Context, organizationRoleTemplate *corev1alpha1.OrganizationRoleTemplate) error {
	// Reconcile Roles, RoleBindings for organizations.
	if organizationRoleTemplate.HasScope(corev1alpha1.RoleTemplateScopeOrganization) {
		readyOrganizations, err := r.listReadyOrganizations(ctx)
		if err != nil {
			return fmt.Errorf("list ready Organizations: %w", err)
		}

		for _, organization := range readyOrganizations {
			// Reconcile Roles and RoleBindings in Organization's namespace.
			if err := r.reconcileRBACForOrganization(ctx, organizationRoleTemplate, organization); err != nil {
				return fmt.Errorf("reconciling RBAC for organization %s:%w", organization.Name, err)
			}
		}
	}
	return nil
}

func (r *OrganizationRoleTemplateReconciler) reconcileRBACForOrganization(ctx context.Context, organizationRoleTemplate *corev1alpha1.OrganizationRoleTemplate, organization *corev1alpha1.Organization) error {
	// Reconcile Role.
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      organizationRoleTemplate.Name,
			Namespace: organization.Status.Namespace.Name,
		},
		Rules: organizationRoleTemplate.Spec.Rules,
	}
	if err := r.reconcileRole(ctx, role, organizationRoleTemplate); err != nil {
		return err
	}

	// Reconcile RoleBindings.
	if organizationRoleTemplate.HasBindTo(corev1alpha1.BindToOwners) {
		roleBinding := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      organizationRoleTemplate.Name,
				Namespace: organization.Status.Namespace.Name,
			},
			Subjects: organization.Spec.Owners,
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     role.Name,
			},
		}
		if err := r.reconcileRoleBinding(ctx, roleBinding, organizationRoleTemplate); err != nil {
			return err
		}
	}
	return nil
}

func (r *OrganizationRoleTemplateReconciler) reconcileRole(ctx context.Context, role *rbacv1.Role, organizationRoleTemplate *corev1alpha1.OrganizationRoleTemplate) error {
	desiredRole := role.DeepCopy()
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, role, func() error {
		if err := controllerutil.SetControllerReference(
			organizationRoleTemplate, role, r.Scheme); err != nil {
			return fmt.Errorf("set controller reference for Role: %w", err)
		}
		if !reflect.DeepEqual(role.Rules, desiredRole.Rules) {
			role.Rules = desiredRole.Rules
		}
		return nil
	}); err != nil {
		return fmt.Errorf("creating or updating Role: %w", err)
	}
	return nil
}

func (r *OrganizationRoleTemplateReconciler) reconcileRoleBinding(ctx context.Context, roleBinding *rbacv1.RoleBinding, organizationRoleTemplate *corev1alpha1.OrganizationRoleTemplate) error {
	desiredRoleBinding := roleBinding.DeepCopy()
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, roleBinding, func() error {
		if err := controllerutil.SetControllerReference(
			organizationRoleTemplate, roleBinding, r.Scheme); err != nil {
			return fmt.Errorf("set controller reference for RoleBinding: %w", err)
		}
		if !reflect.DeepEqual(roleBinding.Subjects, desiredRoleBinding.Subjects) {
			roleBinding.Subjects = desiredRoleBinding.Subjects
		}
		return nil
	}); err != nil {
		return fmt.Errorf("creating or updating RoleBinding: %w", err)
	}
	return nil
}

func (r *OrganizationRoleTemplateReconciler) listReadyOrganizations(ctx context.Context) ([]*corev1alpha1.Organization, error) {
	organizationList := &corev1alpha1.OrganizationList{}
	if err := r.Client.List(ctx, organizationList); err != nil {
		return nil, fmt.Errorf("listing Organizations: %w", err)
	}
	var readyOrganizations []*corev1alpha1.Organization
	for _, organization := range organizationList.Items {
		if organization.IsReady() {
			readyOrganizations = append(readyOrganizations, &organization)
		}
	}
	return readyOrganizations, nil
}
