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
	"k8c.io/utils/pkg/owner"
	"k8c.io/utils/pkg/util"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	corev1alpha1 "k8c.io/bulward/pkg/apis/core/v1alpha1"
	storagev1alpha1 "k8c.io/bulward/pkg/apis/storage/v1alpha1"
)

const (
	projectRoleTemplateControllerFinalizer string = "projectroletemplate.bulward.io/controller"
)

// ProjectRoleTemplateReconciler reconciles a Project object
type ProjectRoleTemplateReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=bulward.io,resources=projectroletemplates,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=bulward.io,resources=projectroletemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=storage.bulward.io,resources=organizations,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=storage.bulward.io,resources=projects,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete;bind;escalate
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete

func (r *ProjectRoleTemplateReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

	projectRoleTemplate := &corev1alpha1.ProjectRoleTemplate{}
	if err := r.Get(ctx, req.NamespacedName, projectRoleTemplate); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !projectRoleTemplate.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, projectRoleTemplate); err != nil {
			return ctrl.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return ctrl.Result{}, nil
	}

	if util.AddFinalizer(projectRoleTemplate, projectRoleTemplateControllerFinalizer) {
		if err := r.Update(ctx, projectRoleTemplate); err != nil {
			return ctrl.Result{}, fmt.Errorf("updating finalizers: %w", err)
		}
	}

	var targets []corev1alpha1.RoleTemplateTarget
	selectedReadyProjects, err := r.listSelectedReadyProjects(ctx, projectRoleTemplate)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("listing selected ready Projects: %w", err)
	}

	// Reconcile Role/RoleBindings in Project namespaces.
	for _, project := range selectedReadyProjects {
		if err := r.reconcileRBACForProject(ctx, projectRoleTemplate, &project); err != nil {
			return ctrl.Result{}, fmt.Errorf("reconcling Project RBAC: %w", err)
		}
		targets = append(targets, corev1alpha1.RoleTemplateTarget{
			Kind:               project.Kind,
			APIGroup:           project.GroupVersionKind().Group,
			Name:               project.Name,
			ObservedGeneration: project.Status.ObservedGeneration,
		})
	}

	var changed bool
	if !reflect.DeepEqual(targets, projectRoleTemplate.Status.Targets) {
		projectRoleTemplate.Status.Targets = targets
		changed = true
	}
	if !projectRoleTemplate.IsReady() {
		// Update ProjectRoleTemplate Status
		projectRoleTemplate.Status.ObservedGeneration = projectRoleTemplate.Generation
		projectRoleTemplate.Status.SetCondition(corev1alpha1.ProjectRoleTemplateCondition{
			Type:    corev1alpha1.ProjectRoleTemplateReady,
			Status:  corev1alpha1.ConditionTrue,
			Reason:  "SetupComplete",
			Message: "ProjectRoleTemplate setup is complete.",
		})
		changed = true
	}

	if changed {
		if err := r.Status().Update(ctx, projectRoleTemplate); err != nil {
			return ctrl.Result{}, fmt.Errorf("updating ProjectRoleTemplate status: %w", err)
		}
	}
	return ctrl.Result{}, nil
}

func (r *ProjectRoleTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	enqueueAllTemplates := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(mapObject handler.MapObject) (out []ctrl.Request) {
			templates := &corev1alpha1.ProjectRoleTemplateList{}
			if err := r.Client.List(context.Background(), templates, client.InNamespace(mapObject.Meta.GetNamespace())); err != nil {
				// This will makes the manager crashes, and it will restart and reconcile all objects again.
				panic(fmt.Errorf("listting ProjectRoleTemplate: %w", err))
			}
			for _, template := range templates.Items {
				out = append(out, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      template.Name,
						Namespace: template.Namespace,
					},
				})
			}
			return
		}),
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.ProjectRoleTemplate{}).
		Watches(&source.Kind{Type: &storagev1alpha1.Project{}}, enqueueAllTemplates).
		Complete(r)
}

// handleDeletion handles the deletion of the ProjectRoleTemplate object:
func (r *ProjectRoleTemplateReconciler) handleDeletion(ctx context.Context, projectRoleTemplate *corev1alpha1.ProjectRoleTemplate) error {
	// Update the ProjectRoleTemplate Status to Terminating.
	readyCondition, _ := projectRoleTemplate.Status.GetCondition(corev1alpha1.ProjectRoleTemplateReady)
	if readyCondition.Status != corev1alpha1.ConditionFalse ||
		readyCondition.Status == corev1alpha1.ConditionFalse && readyCondition.Reason != corev1alpha1.ProjectRoleTemplateTerminatingReason {
		projectRoleTemplate.Status.ObservedGeneration = projectRoleTemplate.Generation
		projectRoleTemplate.Status.SetCondition(corev1alpha1.ProjectRoleTemplateCondition{
			Type:    corev1alpha1.ProjectRoleTemplateReady,
			Status:  corev1alpha1.ConditionFalse,
			Reason:  corev1alpha1.ProjectRoleTemplateTerminatingReason,
			Message: "ProjectRoleTemplate is being terminated",
		})
		if err := r.Status().Update(ctx, projectRoleTemplate); err != nil {
			return fmt.Errorf("updating ProjectRoleTemplate status: %w", err)
		}
	}

	cleanedUp, err := util.DeleteObjects(ctx, r.Client, r.Scheme, []runtime.Object{
		&rbacv1.Role{},
		&rbacv1.RoleBinding{},
	}, owner.OwnedBy(projectRoleTemplate, r.Scheme))
	if err != nil {
		return fmt.Errorf("DeleteObjects: %w", err)
	}
	if cleanedUp && util.RemoveFinalizer(projectRoleTemplate, projectRoleTemplateControllerFinalizer) {
		if err := r.Update(ctx, projectRoleTemplate); err != nil {
			return fmt.Errorf("updating ProjectRoleTemplate Status: %w", err)
		}
	}
	return nil
}

func (r *ProjectRoleTemplateReconciler) reconcileRBACForProject(ctx context.Context, projectRoleTemplate *corev1alpha1.ProjectRoleTemplate, project *storagev1alpha1.Project) error {
	// Reconcile Role.
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      projectRoleTemplate.Name,
			Namespace: project.Status.Namespace.Name,
		},
		Rules: projectRoleTemplate.Spec.Rules,
	}
	if err := r.reconcileRole(ctx, role, projectRoleTemplate); err != nil {
		return err
	}

	// Reconcile RoleBindings.
	var subjects []rbacv1.Subject
	if projectRoleTemplate.HasBinding(corev1alpha1.BindToEveryone) {
		// This is needed, because it can be the case that Organization Owner has not created any RoleBindings for Project
		// Owner, so Project owner will not present in the project.Status.Member.
		subjects = append(subjects, project.Spec.Owners...)
		subjects = append(subjects, project.Status.Members...)
	} else if projectRoleTemplate.HasBinding(corev1alpha1.BindToOwners) {
		subjects = append(subjects, project.Spec.Owners...)
	}
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      projectRoleTemplate.Name,
			Namespace: project.Status.Namespace.Name,
		},
		Subjects: extractSubjects(subjects),
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     role.Name,
		},
	}
	if err := r.reconcileRoleBinding(ctx, roleBinding, projectRoleTemplate); err != nil {
		return err
	}
	return nil
}

func (r *ProjectRoleTemplateReconciler) reconcileRole(ctx context.Context, role *rbacv1.Role, projectRoleTemplate *corev1alpha1.ProjectRoleTemplate) error {
	if _, err := owner.ReconcileOwnedObjects(ctx, r.Client, r.Log, r.Scheme,
		projectRoleTemplate,
		[]runtime.Object{role}, &rbacv1.Role{},
		func(actual, desired runtime.Object) error {
			actualRule := actual.(*rbacv1.Role)
			desiredRole := desired.(*rbacv1.Role)
			actualRule.Rules = desiredRole.Rules
			return nil
		}); err != nil {
		return fmt.Errorf("cannot reconcile Role: %w", err)
	}
	return nil
}

func (r *ProjectRoleTemplateReconciler) reconcileRoleBinding(ctx context.Context, roleBinding *rbacv1.RoleBinding, projectRoleTemplate *corev1alpha1.ProjectRoleTemplate) error {
	if _, err := owner.ReconcileOwnedObjects(ctx, r.Client, r.Log, r.Scheme,
		projectRoleTemplate,
		[]runtime.Object{roleBinding}, &rbacv1.RoleBinding{},
		func(actual, desired runtime.Object) error {
			actualRuleBinding := actual.(*rbacv1.RoleBinding)
			desiredRoleBinding := desired.(*rbacv1.RoleBinding)
			actualRuleBinding.RoleRef = desiredRoleBinding.RoleRef
			actualRuleBinding.Subjects = desiredRoleBinding.Subjects
			return nil
		}); err != nil {
		return fmt.Errorf("cannot reconcile RoleBinding: %w", err)
	}
	return nil
}

func (r *ProjectRoleTemplateReconciler) listSelectedReadyProjects(ctx context.Context, projectRoleTemplate *corev1alpha1.ProjectRoleTemplate) ([]storagev1alpha1.Project, error) {
	projectSelector, err := metav1.LabelSelectorAsSelector(projectRoleTemplate.Spec.ProjectSelector)
	if err != nil {
		return nil, fmt.Errorf("parsing Project selector: %w", err)
	}
	projects := &storagev1alpha1.ProjectList{}
	if err := r.List(ctx, projects, client.InNamespace(projectRoleTemplate.Namespace), client.MatchingLabelsSelector{Selector: projectSelector}); err != nil {
		return nil, fmt.Errorf("listing Project: %w", err)
	}
	var readyProjects []storagev1alpha1.Project
	for _, project := range projects.Items {
		if project.IsReady() {
			readyProjects = append(readyProjects, project)
		}
	}
	return readyProjects, nil
}
