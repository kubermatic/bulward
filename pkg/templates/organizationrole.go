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

package templates

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
)

const (
	ProjectAdminOrganizationRoleTemplateName = "project-admin"
	RBACAdminOrganizationRoleTemplateName    = "rbac-admin"
)

func DefaultOrganizationRoleTemplatesForOwners() []*corev1alpha1.OrganizationRoleTemplate {
	return []*corev1alpha1.OrganizationRoleTemplate{ProjectAdminOrganizationRoleTemplate(), RBACAdminOrganizationRoleTemplate()}
}

func ProjectAdminOrganizationRoleTemplate() *corev1alpha1.OrganizationRoleTemplate {
	return &corev1alpha1.OrganizationRoleTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: ProjectAdminOrganizationRoleTemplateName,
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
					Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
				},
			},
		},
	}
}

func RBACAdminOrganizationRoleTemplate() *corev1alpha1.OrganizationRoleTemplate {
	return &corev1alpha1.OrganizationRoleTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: RBACAdminOrganizationRoleTemplateName,
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
