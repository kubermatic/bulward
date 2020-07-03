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
	"sort"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/kubermatic/utils/pkg/owner"
	"github.com/kubermatic/utils/pkg/util"

	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
)

const (
	organizationControllerFinalizer string = "organization.bulward.io/controller"
)

// OrganizationReconciler reconciles a Organization object
type OrganizationReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=bulward.io,resources=organizations,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=bulward.io,resources=organizations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=bulward.io,resources=organizationroletemplates,verbs=create
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch

// Reconcile function reconciles the Organization object which specified by the request. Currently, it does the following:
// 1. Fetch the Organization object.
// 2. Handle the deletion of the Organization object (Remove the namespace that the Organization owns, and remove the finalizer).
// 3. Handle the creation/update of the Organization object (Create/reconcile the namespace and insert the finalizer).
// 4. Create project-admin and rbac-admin OrganizationRoleTemplate for owners of the Organization.
// 5. Update the status of the Organization object.
func (r *OrganizationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("Organization", req.NamespacedName)

	organization := &corev1alpha1.Organization{}
	if err := r.Get(ctx, req.NamespacedName, organization); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !organization.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, organization); err != nil {
			return ctrl.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return ctrl.Result{}, nil
	}

	if util.AddFinalizer(organization, organizationControllerFinalizer) {
		if err := r.Update(ctx, organization); err != nil {
			return ctrl.Result{}, fmt.Errorf("updating finalizers: %w", err)
		}
	}

	if err := r.reconcileNamespace(ctx, log, organization); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling namespace: %w", err)
	}
	if err := r.reconcileMembers(ctx, log, organization); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling members: %w", err)
	}

	if err := r.createDefaultOrganizationRoleTemplates(ctx); err != nil {
		return ctrl.Result{}, fmt.Errorf("creating default OrganizationRoleTemplates: %w", err)
	}

	if !organization.IsReady() {
		organization.Status.ObservedGeneration = organization.Generation
		organization.Status.SetCondition(corev1alpha1.OrganizationCondition{
			Type:    corev1alpha1.OrganizationReady,
			Status:  corev1alpha1.ConditionTrue,
			Reason:  "SetupComplete",
			Message: "Organization setup is complete.",
		})
		if err := r.Status().Update(ctx, organization); err != nil {
			return ctrl.Result{}, fmt.Errorf("updating Organization status: %w", err)
		}
	}
	return ctrl.Result{}, nil
}

func (r *OrganizationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	enqueuer := owner.EnqueueRequestForOwner(&corev1alpha1.Organization{}, mgr.GetScheme())

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Organization{}).
		Watches(&source.Kind{Type: &corev1.Namespace{}}, enqueuer).
		Watches(&source.Kind{Type: &rbacv1.RoleBinding{}}, &handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(object handler.MapObject) []reconcile.Request {
				return []reconcile.Request{{
					// Organization name is its namespace
					NamespacedName: types.NamespacedName{Name: object.Meta.GetNamespace()},
				}}
			}),
		}).
		Complete(r)
}

// handleDeletion handles the deletion of the Organization object:
func (r *OrganizationReconciler) handleDeletion(ctx context.Context, log logr.Logger, organization *corev1alpha1.Organization) error {
	// Update the Organization Status to Terminating.
	readyCondition, _ := organization.Status.GetCondition(corev1alpha1.OrganizationReady)
	if readyCondition.Status != corev1alpha1.ConditionFalse ||
		readyCondition.Status == corev1alpha1.ConditionFalse && readyCondition.Reason != corev1alpha1.OrganizationTerminatingReason {
		organization.Status.ObservedGeneration = organization.Generation
		organization.Status.SetCondition(corev1alpha1.OrganizationCondition{
			Type:    corev1alpha1.OrganizationReady,
			Status:  corev1alpha1.ConditionFalse,
			Reason:  corev1alpha1.OrganizationTerminatingReason,
			Message: "Organization is being terminated",
		})
		if err := r.Status().Update(ctx, organization); err != nil {
			return fmt.Errorf("updating Organization status: %w", err)
		}
	}

	cleanedUp, err := util.DeleteObjects(ctx, r.Client, r.Scheme, []runtime.Object{
		&corev1.Namespace{},
	}, owner.OwnedBy(organization, r.Scheme))
	if err != nil {
		return fmt.Errorf("DeleteObjects: %w", err)
	}
	if cleanedUp && util.RemoveFinalizer(organization, organizationControllerFinalizer) {
		if err := r.Update(ctx, organization); err != nil {
			return fmt.Errorf("updating Organization Status: %w", err)
		}
	}
	return nil
}

func (r *OrganizationReconciler) reconcileNamespace(ctx context.Context, log logr.Logger, organization *corev1alpha1.Organization) error {
	ns := &corev1.Namespace{}
	ns.Name = organization.Name

	if _, err := owner.ReconcileOwnedObjects(ctx, r.Client, log, r.Scheme, organization, []runtime.Object{ns}, &corev1.Namespace{}, nil); err != nil {
		return fmt.Errorf("cannot reconcile namespace: %w", err)
	}

	if organization.Status.Namespace == nil {
		organization.Status.Namespace = &corev1alpha1.ObjectReference{
			Name: ns.Name,
		}
		if err := r.Status().Update(ctx, organization); err != nil {
			return fmt.Errorf("updating NamespaceName: %w", err)
		}
	}
	return nil
}

func (r *OrganizationReconciler) reconcileMembers(ctx context.Context, log logr.Logger, organization *corev1alpha1.Organization) error {
	rbs := &rbacv1.RoleBindingList{}
	if err := r.List(ctx, rbs, client.InNamespace(organization.Status.Namespace.Name)); err != nil {
		return fmt.Errorf("list rolebindings: %w", err)
	}
	organization.Status.Members = r.extractSubjects(rbs)
	if err := r.Status().Update(ctx, organization); err != nil {
		return fmt.Errorf("updating members: %w", err)
	}
	return nil
}

func (r *OrganizationReconciler) extractSubjects(rbs *rbacv1.RoleBindingList) []rbacv1.Subject {
	var subjects []rbacv1.Subject
	for _, rb := range rbs.Items {
		subjects = append(subjects, rb.Subjects...)
	}
	sort.Slice(subjects, func(i, j int) bool {
		a := subjects[i]
		b := subjects[j]
		return a.String() < b.String()
	})
	filteredSubjects := make([]rbacv1.Subject, 0, len(subjects))
	for i := range subjects {
		if i == 0 || subjects[i-1].String() != subjects[i].String() {
			filteredSubjects = append(filteredSubjects, subjects[i])
		}
	}
	return filteredSubjects
}
func (r *OrganizationReconciler) createDefaultOrganizationRoleTemplates(ctx context.Context) error {
	desiredProjectAdminOrganizationRoleTemplate := r.buildProjectAdminOrganizationRoleTemplate()
	desiredRBACAdminOrganizationRoleTemplate := r.buildRBACAdminOrganizationRoleTemplate()

	if err := r.createOrganizationRoleTemplate(ctx, desiredProjectAdminOrganizationRoleTemplate); err != nil {
		return fmt.Errorf("creating project-admin OrganizationRoleTemplate: %w", err)
	}

	if err := r.createOrganizationRoleTemplate(ctx, desiredRBACAdminOrganizationRoleTemplate); err != nil {
		return fmt.Errorf("creating rbac-admin OrganizationRoleTemplate: %w", err)
	}
	return nil
}

func (r *OrganizationReconciler) createOrganizationRoleTemplate(ctx context.Context,
	desiredOrganizationRoleTemplate *corev1alpha1.OrganizationRoleTemplate,
) error {
	if err := r.Client.Create(ctx, desiredOrganizationRoleTemplate); err != nil {
		if errors.IsAlreadyExists(err) {
			// default OrganizationRoleTemplate is created by some other Organizations, no need to create again.
			return nil
		}
		return err
	}
	// default OrganizationRoleTemplate is created.
	return nil
}

func (r *OrganizationReconciler) buildProjectAdminOrganizationRoleTemplate() *corev1alpha1.OrganizationRoleTemplate {
	return &corev1alpha1.OrganizationRoleTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: "project-admin",
		},
		Spec: corev1alpha1.OrganizationRoleTemplateSpec{
			Scopes: []corev1alpha1.RoleTemplateScope{
				corev1alpha1.RoleTemplateScopeOrganization,
			},
			BindTo: []corev1alpha1.BindToType{
				corev1alpha1.BindToOwners,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"apiserver.bulward.io"},
					Resources: []string{"projects"},
					Verbs:     []string{rbacv1.VerbAll},
				},
			},
		},
	}
}

func (r *OrganizationReconciler) buildRBACAdminOrganizationRoleTemplate() *corev1alpha1.OrganizationRoleTemplate {
	return &corev1alpha1.OrganizationRoleTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rbac-admin",
		},
		Spec: corev1alpha1.OrganizationRoleTemplateSpec{
			Scopes: []corev1alpha1.RoleTemplateScope{
				corev1alpha1.RoleTemplateScopeOrganization,
			},
			BindTo: []corev1alpha1.BindToType{
				corev1alpha1.BindToOwners,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"rbac.authorization.k8s.io"},
					Resources: []string{"roles"},
					Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete", "bind"},
				},
				{
					APIGroups: []string{"rbac.authorization.k8s.io"},
					Resources: []string{"rolebindings"},
					Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
				},
			},
		},
	}
}
