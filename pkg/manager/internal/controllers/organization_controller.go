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
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/kubermatic/utils/pkg/owner"
	"github.com/kubermatic/utils/pkg/util"

	storagev1alpha1 "github.com/kubermatic/bulward/pkg/apis/storage/v1alpha1"
	"github.com/kubermatic/bulward/pkg/templates"
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

// +kubebuilder:rbac:groups=storage.bulward.io,resources=organizations,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=storage.bulward.io,resources=organizations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=storage.bulward.io,resources=projects,verbs=get;list;watch
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

	organization := &storagev1alpha1.Organization{}
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
	if err := r.checkOrganizationRoleTemplatesForOwners(ctx); err != nil {
		return ctrl.Result{}, fmt.Errorf("checking default OrganizationRoleTemplates: %w", err)
	}

	if !organization.IsReady() {
		organization.Status.ObservedGeneration = organization.Generation
		organization.Status.SetCondition(storagev1alpha1.OrganizationCondition{
			Type:    storagev1alpha1.OrganizationReady,
			Status:  storagev1alpha1.ConditionTrue,
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
	enqueuerForOwner := owner.EnqueueRequestForOwner(&storagev1alpha1.Organization{}, mgr.GetScheme())
	enqueuerByNamespace := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(object handler.MapObject) []reconcile.Request {
			return []reconcile.Request{{
				// Organization name is its namespace
				NamespacedName: types.NamespacedName{Name: object.Meta.GetNamespace()},
			}}
		}),
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&storagev1alpha1.Organization{}).
		Watches(&source.Kind{Type: &corev1.Namespace{}}, enqueuerForOwner).
		Watches(&source.Kind{Type: &rbacv1.RoleBinding{}}, enqueuerByNamespace).
		Watches(&source.Kind{Type: &storagev1alpha1.Project{}}, enqueuerByNamespace).
		Complete(r)
}

// handleDeletion handles the deletion of the Organization object:
func (r *OrganizationReconciler) handleDeletion(ctx context.Context, log logr.Logger, organization *storagev1alpha1.Organization) error {
	// Update the Organization Status to Terminating.
	readyCondition, _ := organization.Status.GetCondition(storagev1alpha1.OrganizationReady)
	if readyCondition.Status != storagev1alpha1.ConditionFalse ||
		readyCondition.Status == storagev1alpha1.ConditionFalse && readyCondition.Reason != storagev1alpha1.OrganizationTerminatingReason {
		organization.Status.ObservedGeneration = organization.Generation
		organization.Status.SetCondition(storagev1alpha1.OrganizationCondition{
			Type:    storagev1alpha1.OrganizationReady,
			Status:  storagev1alpha1.ConditionFalse,
			Reason:  storagev1alpha1.OrganizationTerminatingReason,
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

func (r *OrganizationReconciler) reconcileNamespace(ctx context.Context, log logr.Logger, organization *storagev1alpha1.Organization) error {
	ns := &corev1.Namespace{}
	ns.Name = organization.Name

	if _, err := owner.ReconcileOwnedObjects(ctx, r.Client, log, r.Scheme, organization, []runtime.Object{ns}, &corev1.Namespace{}, nil); err != nil {
		return fmt.Errorf("cannot reconcile namespace: %w", err)
	}

	if organization.Status.Namespace == nil {
		organization.Status.Namespace = &storagev1alpha1.ObjectReference{
			Name: ns.Name,
		}
		if err := r.Status().Update(ctx, organization); err != nil {
			return fmt.Errorf("updating NamespaceName: %w", err)
		}
	}
	return nil
}

func (r *OrganizationReconciler) reconcileMembers(ctx context.Context, log logr.Logger, organization *storagev1alpha1.Organization) error {
	var subjects []rbacv1.Subject
	rbs := &rbacv1.RoleBindingList{}
	if err := r.List(ctx, rbs, client.InNamespace(organization.Status.Namespace.Name)); err != nil {
		return fmt.Errorf("list rolebindings: %w", err)
	}
	for _, roleBinding := range rbs.Items {
		subjects = append(subjects, roleBinding.Subjects...)
	}
	// Propagate members of Project under this Organization.
	projects := &storagev1alpha1.ProjectList{}
	if err := r.List(ctx, projects, client.InNamespace(organization.Status.Namespace.Name)); err != nil {
		return fmt.Errorf("list Projects: %w", err)
	}
	for _, project := range projects.Items {
		if project.IsReady() {
			subjects = append(subjects, project.Status.Members...)
		}
	}
	organization.Status.Members = extractSubjects(subjects)
	if err := r.Status().Update(ctx, organization); err != nil {
		return fmt.Errorf("updating members: %w", err)
	}
	return nil
}

// checkOrganizationRoleTemplatesForOwners checks if the bulward pre-defined OrganizationRoleTemplates for Organization owners
// (project-admin, rbac-admin) are present. If not, just create them.
func (r *OrganizationReconciler) checkOrganizationRoleTemplatesForOwners(ctx context.Context) error {
	ownerTemplates := templates.DefaultOrganizationRoleTemplatesForOwners()

	for _, template := range ownerTemplates {
		if err := r.Client.Create(ctx, template); err != nil {
			if errors.IsAlreadyExists(err) {
				// default OrganizationRoleTemplate is created by some other Organizations, and it will be shared across
				// organizations in the system, so no need to create again.
				continue
			}
			return fmt.Errorf("creating owner OrganizationRoleTemplate: %s: %w", template.Name, err)
		}
	}
	return nil
}
