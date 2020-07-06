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
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/kubermatic/utils/pkg/testutil"

	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
	"github.com/kubermatic/bulward/pkg/templates"
)

func init() {
	utilruntime.Must(corev1alpha1.AddToScheme(testScheme))
	utilruntime.Must(scheme.AddToScheme(testScheme))
}

func TestCoreOrganization(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cfg, err := controllerruntime.GetConfig()
	require.NoError(t, err)
	cl := testutil.NewRecordingClient(t, cfg, testScheme, testutil.CleanupOnSuccess)
	t.Cleanup(cl.CleanUpFunc(ctx))

	owner := rbacv1.Subject{
		Kind:     "User",
		APIGroup: "rbac.authorization.k8s.io",
		Name:     "Owner1",
	}

	org := &corev1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name: "core-organization-test",
		},
		Spec: corev1alpha1.OrganizationSpec{
			Metadata: &corev1alpha1.OrganizationMetadata{
				DisplayName: "berlin",
				Description: "a humble organization of German capital",
			},
			Owners: []rbacv1.Subject{owner},
		},
	}
	require.NoError(t, testutil.DeleteAndWaitUntilNotFound(ctx, cl, org))
	require.NoError(t, cl.Create(ctx, org))
	require.NoError(t, testutil.WaitUntilReady(ctx, cl, org))

	projectTemplate := &corev1alpha1.OrganizationRoleTemplate{}
	require.NoError(t, cl.Get(ctx, types.NamespacedName{
		Name: templates.ProjectAdminOrganizationRoleTemplateName,
	}, projectTemplate))

	rbacTemplate := &corev1alpha1.OrganizationRoleTemplate{}
	require.NoError(t, cl.Get(ctx, types.NamespacedName{
		Name: templates.RBACAdminOrganizationRoleTemplateName,
	}, rbacTemplate))

	assert.Empty(t, org.Status.Members)
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
			Name:     "dummy",
		},
	}
	require.NoError(t, cl.Create(ctx, rb))
	require.NoError(t, cl.WaitUntil(ctx, org, func() (done bool, err error) {
		if len(org.Status.Members) > 0 {
			assert.Equal(t, []rbacv1.Subject{rbacSubject}, org.Status.Members)
			return true, nil
		}
		return false, nil
	}))
	require.NoError(t, testutil.DeleteAndWaitUntilNotFound(ctx, cl, rb))
	require.NoError(t, cl.WaitUntil(ctx, org, func() (done bool, err error) {
		return len(org.Status.Members) == 0, nil
	}), "organization never reached 0 memebers")
}
