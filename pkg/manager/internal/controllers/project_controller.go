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
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	storagev1alpha1 "github.com/kubermatic/bulward/pkg/apis/storage/v1alpha1"
	"github.com/kubermatic/utils/pkg/owner"
)

const projectNamespaceNameSeparator = "-bulward-"

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=storage.bulward.io,resources=projects,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=storage.bulward.io,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch

func (r *ProjectReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("Project", req.NamespacedName)

	project := &storagev1alpha1.Project{}
	if err := r.Get(ctx, req.NamespacedName, project); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.reconcileNamespace(ctx, log, project); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling namespace: %w", err)
	}

	if err := r.reconcileMembers(ctx, project); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling members: %w", err)
	}

	if !project.IsReady() {
		project.Status.ObservedGeneration = project.Generation
		project.Status.SetCondition(storagev1alpha1.ProjectCondition{
			Type:    storagev1alpha1.ProjectReady,
			Status:  storagev1alpha1.ConditionTrue,
			Reason:  "SetupComplete",
			Message: "Project setup is complete.",
		})
		if err := r.Status().Update(ctx, project); err != nil {
			return ctrl.Result{}, fmt.Errorf("updating Projects status: %w", err)
		}
	}
	return ctrl.Result{}, nil
}

func (r *ProjectReconciler) reconcileNamespace(ctx context.Context, log logr.Logger, project *storagev1alpha1.Project) error {
	ns := &corev1.Namespace{}
	ns.Name = generateProjectNamespaceName(project.Namespace, project.Name)

	if _, err := owner.ReconcileOwnedObjects(ctx, r.Client, log, r.Scheme, project, []runtime.Object{ns}, &corev1.Namespace{}, nil); err != nil {
		return fmt.Errorf("cannot reconcile namespace: %w", err)
	}

	if project.Status.Namespace == nil {
		project.Status.Namespace = &storagev1alpha1.ObjectReference{
			Name: ns.Name,
		}
		if err := r.Status().Update(ctx, project); err != nil {
			return fmt.Errorf("updating NamespaceName: %w", err)
		}
	}

	return nil
}

func (r *ProjectReconciler) reconcileMembers(ctx context.Context, project *storagev1alpha1.Project) error {
	rbs := &rbacv1.RoleBindingList{}
	if err := r.List(ctx, rbs, client.InNamespace(project.Status.Namespace.Name)); err != nil {
		return fmt.Errorf("list rolebindings: %w", err)
	}
	project.Status.Members = extractSubjects(rbs)
	if err := r.Status().Update(ctx, project); err != nil {
		return fmt.Errorf("updating members: %w", err)
	}
	return nil
}

// Project Namespace name is organization_name-separator-project_name
func generateProjectNamespaceName(organizatioName, projectName string) string {
	return fmt.Sprintf("%s%s%s", organizatioName, projectNamespaceNameSeparator, projectName)
}

// Project Namespace name is organization_name-separator-project_name
func deconstructProjectNamespaceName(projectNamespaceName string) (string, string, error) {
	split := strings.Split(projectNamespaceName, projectNamespaceNameSeparator)
	if len(split) != 2 {
		return "", "", fmt.Errorf("not a valid project namespace name to deconstruct: %s", projectNamespaceName)
	}

	return split[0], split[1], nil
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	enqueuer := owner.EnqueueRequestForOwner(&storagev1alpha1.Project{}, mgr.GetScheme())

	return ctrl.NewControllerManagedBy(mgr).
		For(&storagev1alpha1.Project{}).
		Watches(&source.Kind{Type: &corev1.Namespace{}}, enqueuer).
		Watches(&source.Kind{Type: &rbacv1.RoleBinding{}}, &handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(object handler.MapObject) []reconcile.Request {
				org, pro, err := deconstructProjectNamespaceName(object.Meta.GetNamespace())
				if err != nil {
					return []reconcile.Request{}
				}
				return []reconcile.Request{{
					NamespacedName: types.NamespacedName{
						Name:      pro,
						Namespace: org,
					},
				}}
			}),
		}).
		Complete(r)
}
