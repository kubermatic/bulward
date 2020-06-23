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
	"github.com/kubermatic/bulward/pkg/internal/owner"
	"github.com/kubermatic/bulward/pkg/internal/util"
)

const (
	internalOrganizationControllerFinalizer string = "internalorganization.bulward.io/controller"
)

// InternalOrganizationReconciler reconciles a InternalOrganization object
type InternalOrganizationReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=bulward.io,resources=internalorganizations,verbs=get;list;watch;update;
// +kubebuilder:rbac:groups=bulward.io,resources=internalorganizations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// Reconcile function reconciles the InternalOrganization object which specified by the request. Currently, it does the following:
// 1. Fetch the InternalOrganization object.
// 2. Handle the deletion of the InternalOrganization object (Remove the namespace that the internalOrganization owns, and remove the finalizer).
// 3. Handle the creation/update of the InternalOrganization object (Create/reconcile the namespace and insert the finalizer).
// 4. Update the status of the InternalOrganization object.
func (r *InternalOrganizationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("InternalOrganization", req.NamespacedName)

	internalOrganization := &corev1alpha1.InternalOrganization{}
	if err := r.Get(ctx, req.NamespacedName, internalOrganization); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !internalOrganization.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, internalOrganization); err != nil {
			return ctrl.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return ctrl.Result{}, nil
	}

	if util.AddFinalizer(internalOrganization, internalOrganizationControllerFinalizer) {
		if err := r.Update(ctx, internalOrganization); err != nil {
			return ctrl.Result{}, fmt.Errorf("updating finalizers: %w", err)
		}
	}
	if err := r.reconcileNamespace(ctx, log, internalOrganization); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling namespace: %w", err)
	}

	if !internalOrganization.IsReady() {
		// Update InternalOrganization Status
		internalOrganization.Status.ObservedGeneration = internalOrganization.Generation
		internalOrganization.Status.SetCondition(corev1alpha1.InternalOrganizationCondition{
			Type:    corev1alpha1.InternalOrganizationReady,
			Status:  corev1alpha1.ConditionTrue,
			Reason:  "SetupComplete",
			Message: "InternalOrganization setup is complete.",
		})
		if err := r.Status().Update(ctx, internalOrganization); err != nil {
			return ctrl.Result{}, fmt.Errorf("updating InternalOrganization status: %w", err)
		}
	}
	return ctrl.Result{}, nil
}

func (r *InternalOrganizationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	enqueuer := owner.EnqueueRequestForOwner(&corev1alpha1.InternalOrganization{}, mgr.GetScheme())

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.InternalOrganization{}).
		Watches(&source.Kind{Type: &corev1.Namespace{}}, enqueuer).
		Complete(r)
}

// handleDeletion handles the deletion of the InternalOrganization object:
func (r *InternalOrganizationReconciler) handleDeletion(ctx context.Context, log logr.Logger, internalOrganization *corev1alpha1.InternalOrganization) error {
	// Update the InternalOrganization Status to Terminating.
	readyCondition, _ := internalOrganization.Status.GetCondition(corev1alpha1.InternalOrganizationReady)
	if readyCondition.Status != corev1alpha1.ConditionFalse ||
		readyCondition.Status == corev1alpha1.ConditionFalse && readyCondition.Reason != corev1alpha1.InternalOrganizationTerminatingReason {
		internalOrganization.Status.ObservedGeneration = internalOrganization.Generation
		internalOrganization.Status.SetCondition(corev1alpha1.InternalOrganizationCondition{
			Type:    corev1alpha1.InternalOrganizationReady,
			Status:  corev1alpha1.ConditionFalse,
			Reason:  corev1alpha1.InternalOrganizationTerminatingReason,
			Message: "InternalOrganization is being terminated",
		})
		if err := r.Status().Update(ctx, internalOrganization); err != nil {
			return fmt.Errorf("updating InternalOrganization status: %w", err)
		}
	}

	cleanedUp, err := util.DeleteObjects(ctx, r.Client, r.Scheme, []runtime.Object{
		&corev1.Namespace{},
	}, owner.OwnedBy(internalOrganization, r.Scheme))
	if err != nil {
		return fmt.Errorf("DeleteObjects: %w", err)
	}
	if cleanedUp && util.RemoveFinalizer(internalOrganization, internalOrganizationControllerFinalizer) {
		if err := r.Update(ctx, internalOrganization); err != nil {
			return fmt.Errorf("updating InternalOrganization Status: %w", err)
		}
	}
	return nil
}

func (r *InternalOrganizationReconciler) reconcileNamespace(ctx context.Context, log logr.Logger, internalOrganization *corev1alpha1.InternalOrganization) error {
	ns := &corev1.Namespace{}
	ns.Name = internalOrganization.Name

	if _, err := owner.ReconcileOwnedObjects(ctx, r.Client, log, r.Scheme, internalOrganization, []runtime.Object{ns}, &corev1.Namespace{}, nil); err != nil {
		return fmt.Errorf("cannot reconcile namespace: %w", err)
	}

	if internalOrganization.Status.Namespace == nil {
		internalOrganization.Status.Namespace = &corev1alpha1.ObjectReference{
			Name: ns.Name,
		}
		if err := r.Status().Update(ctx, internalOrganization); err != nil {
			return fmt.Errorf("updating NamespaceName: %w", err)
		}
	}
	return nil
}
