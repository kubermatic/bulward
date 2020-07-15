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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/kubermatic/utils/pkg/testutil"

	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
	storagev1alpha1 "github.com/kubermatic/bulward/pkg/apis/storage/v1alpha1"
	"github.com/kubermatic/bulward/pkg/templates"
)

func init() {
	utilruntime.Must(corev1alpha1.AddToScheme(testScheme))
	utilruntime.Must(clientgoscheme.AddToScheme(testScheme))
	utilruntime.Must(storagev1alpha1.AddToScheme(testScheme))
}

func TestStorageOrganization(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cfg, err := controllerruntime.GetConfig()
	require.NoError(t, err)
	cl := testutil.NewRecordingClient(t, cfg, testScheme, testutil.CleanUpStrategy(cleanUpStrategy))
	t.Cleanup(cl.CleanUpFunc(ctx))

	owner := rbacv1.Subject{
		Kind:     "User",
		APIGroup: "rbac.authorization.k8s.io",
		Name:     "Owner1",
	}

	org := &storagev1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name: "core-organization-test",
		},
		Spec: storagev1alpha1.OrganizationSpec{
			Metadata: &storagev1alpha1.OrganizationMetadata{
				DisplayName: "berlin",
				Description: "a humble organization of German capital",
			},
			Owners: []rbacv1.Subject{owner},
		},
	}
	require.NoError(t, testutil.DeleteAndWaitUntilNotFound(ctx, cl, org))
	require.NoError(t, cl.Create(ctx, org))
	require.NoError(t, testutil.WaitUntilReady(ctx, cl, org))

	projectTemplate := &corev1alpha1.OrganizationRoleTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: templates.ProjectAdminOrganizationRoleTemplateName,
		},
	}
	require.NoError(t, testutil.WaitUntilReady(ctx, cl, projectTemplate))
	rbacTemplate := &corev1alpha1.OrganizationRoleTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: templates.RBACAdminOrganizationRoleTemplateName,
		},
	}
	require.NoError(t, testutil.WaitUntilReady(ctx, cl, rbacTemplate))
	require.NoError(t, cl.WaitUntil(ctx, org, func() (done bool, err error) {
		if len(org.Status.Members) > 0 {
			assert.Len(t, org.Status.Members, 1)
			assert.Equal(t, org.Status.Members[0].Name, owner.Name)
			return true, nil
		}
		return false, nil
	}))

	t.Log("Organization Owner has permission to create RoleBinding")
	cfg.Impersonate = rest.ImpersonationConfig{
		UserName: owner.Name,
	}
	ownerClient := testutil.NewRecordingClient(t, cfg, testScheme, testutil.CleanUpStrategy(cleanUpStrategy))
	t.Cleanup(ownerClient.CleanUpFunc(ctx))
	rbacSubject := rbacv1.Subject{
		Kind:     "User",
		APIGroup: "rbac.authorization.k8s.io",
		Name:     "User1",
	}
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user1-rb",
			Namespace: org.Status.Namespace.Name,
		},
		Subjects: []rbacv1.Subject{rbacSubject},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     rbacTemplate.Name,
		},
	}
	require.NoError(t, ownerClient.Create(ctx, rb))
	require.NoError(t, cl.WaitUntil(ctx, org, func() (done bool, err error) {
		if len(org.Status.Members) == 2 {
			assert.Contains(t, org.Status.Members, rbacSubject)
			return true, nil
		}
		return false, nil
	}))
	require.NoError(t, testutil.DeleteAndWaitUntilNotFound(ctx, ownerClient, rb))
	require.NoError(t, cl.WaitUntil(ctx, org, func() (done bool, err error) {
		return len(org.Status.Members) == 1, nil
	}), "organization owner can not remove organization member")
}
