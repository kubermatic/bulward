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

package test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8c.io/utils/pkg/testutil"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"

	corev1alpha1 "k8c.io/bulward/pkg/apis/core/v1alpha1"
	storagev1alpha1 "k8c.io/bulward/pkg/apis/storage/v1alpha1"
)

func TestOrganizationRole(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cfg, err := controllerruntime.GetConfig()
	require.NoError(t, err)
	cl := testutil.NewRecordingClient(t, cfg, testScheme, testutil.CleanUpStrategy(cleanUpStrategy))
	t.Cleanup(cl.CleanUpFunc(ctx))
	testName := strings.ToLower(t.Name())

	organizationOwner := rbacv1.Subject{
		Kind:     rbacv1.UserKind,
		APIGroup: rbacv1.GroupName,
		Name:     "Organization Owner",
	}

	org := &storagev1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name: testName,
		},
		Spec: storagev1alpha1.OrganizationSpec{
			Metadata: &storagev1alpha1.OrganizationMetadata{
				DisplayName: "berlin",
				Description: "a humble organization of German capital",
			},
			Owners: []rbacv1.Subject{organizationOwner},
		},
	}
	require.NoError(t, cl.Create(ctx, org))
	require.NoError(t, testutil.WaitUntilReady(ctx, cl, org))

	t.Log("Create an OrganizationRoleTemplate object to test")
	organizationRoleTemplate := &corev1alpha1.OrganizationRoleTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: testName,
		},
		Spec: corev1alpha1.OrganizationRoleTemplateSpec{
			Scopes: []corev1alpha1.RoleTemplateScope{
				corev1alpha1.RoleTemplateScopeOrganization,
				corev1alpha1.RoleTemplateScopeProject,
			},
			BindTo: []corev1alpha1.BindingType{
				corev1alpha1.BindToOwners,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"example.app.io"},
					Resources: []string{"resources", "apps"},
					Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
				},
			},
		},
	}
	require.NoError(t, cl.Create(ctx, organizationRoleTemplate))
	require.NoError(t, testutil.WaitUntilReady(ctx, cl, organizationRoleTemplate))

	t.Log("Check OrganizationRoles and Roles are ready/created.")
	organizationRole := &corev1alpha1.OrganizationRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: org.Status.Namespace.Name,
		},
	}
	require.NoError(t, testutil.WaitUntilReady(ctx, cl, organizationRole))
	testutil.LogObject(t, organizationRole)
	assert.ElementsMatch(t, organizationRoleTemplate.Spec.Rules, organizationRole.Status.AcceptedRules)
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: org.Status.Namespace.Name,
		},
	}
	require.NoError(t, testutil.WaitUntilFound(ctx, cl, role))
	assert.ElementsMatch(t, organizationRoleTemplate.Spec.Rules, role.Rules)

	t.Log("Owner create OrganizationRole object to enable sub-role")
	cfg.Impersonate = rest.ImpersonationConfig{
		UserName: organizationOwner.Name,
	}
	ownerClient := testutil.NewRecordingClient(t, cfg, testScheme, testutil.CleanUpStrategy(cleanUpStrategy))
	viewRole := &corev1alpha1.OrganizationRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-view-role", testName),
			Namespace: org.Status.Namespace.Name,
		},
		Spec: corev1alpha1.OrganizationRoleSpec{
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"example.app.io"},
					Resources: []string{"resources", "apps"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
	}
	require.NoError(t, ownerClient.Create(ctx, viewRole))
	require.NoError(t, testutil.WaitUntilReady(ctx, cl, viewRole))
	role = &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-view-role", testName),
			Namespace: org.Status.Namespace.Name,
		},
	}
	require.NoError(t, testutil.WaitUntilFound(ctx, cl, role))
	assert.ElementsMatch(t, viewRole.Spec.Rules, role.Rules)
	assert.ElementsMatch(t, viewRole.Status.AcceptedRules, role.Rules)

	t.Log("Organization revoke partial permissions")
	organizationRoleTemplate.Spec.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{"example.app.io"},
			Resources: []string{"apps"},
			Verbs:     []string{"get"},
		},
	}
	require.NoError(t, cl.Update(ctx, organizationRoleTemplate))
	require.NoError(t, cl.WaitUntil(ctx, role, func() (done bool, err error) {
		return reflect.DeepEqual(role.Rules, organizationRoleTemplate.Spec.Rules), nil
	}), "subRole is not revoked.")

	t.Log("Clean up")
	require.NoError(t, testutil.DeleteAndWaitUntilNotFound(ctx, cl, organizationRoleTemplate))
	require.NoError(t, cl.WaitUntil(ctx, viewRole, func() (done bool, err error) {
		return len(viewRole.Status.AcceptedRules) == 0, nil
	}), "subRole is not revoked.")
	require.NoError(t, cl.Delete(ctx, viewRole))
	require.NoError(t, cl.WaitUntilNotFound(ctx, role))
}
