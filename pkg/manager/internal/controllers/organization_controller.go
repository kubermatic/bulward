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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"

	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
	"github.com/kubermatic/utils/pkg/owner"
	"github.com/kubermatic/utils/pkg/util"
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

// +kubebuilder:rbac:groups=bulward.io,resources=organizations,verbs=get;list;watch;update;
// +kubebuilder:rbac:groups=bulward.io,resources=organizations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// Reconcile function reconciles the Organization object which specified by the request. Currently, it does the following:
// 1. Fetch the Organization object.
// 2. Handle the deletion of the Organization object (Remove the namespace that the Organization owns, and remove the finalizer).
// 3. Handle the creation/update of the Organization object (Create/reconcile the namespace and insert the finalizer).
// 4. Update the status of the Organization object.
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

	if !organization.IsReady() {
		// Update Organization Status
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
