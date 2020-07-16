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
	"strings"
	"testing"

	"github.com/kubermatic/utils/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"

	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
	storagev1alpha1 "github.com/kubermatic/bulward/pkg/apis/storage/v1alpha1"
	"github.com/kubermatic/bulward/pkg/templates"
)

func init() {
	utilruntime.Must(corev1alpha1.AddToScheme(testScheme))
	utilruntime.Must(clientgoscheme.AddToScheme(testScheme))
	utilruntime.Must(storagev1alpha1.AddToScheme(testScheme))
}

func TestStorageProject(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cfg, err := controllerruntime.GetConfig()
	require.NoError(t, err)
	cl := testutil.NewRecordingClient(t, cfg, testScheme, testutil.CleanUpStrategy(cleanUpStrategy))
	t.Cleanup(cl.CleanUpFunc(ctx))

	organizationOwner := rbacv1.Subject{
		Kind:     rbacv1.UserKind,
		APIGroup: rbacv1.GroupName,
		Name:     "Organization Owner",
	}

	org := &storagev1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name: strings.ToLower(t.Name()),
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

	projectOwner := rbacv1.Subject{
		Kind:     rbacv1.UserKind,
		APIGroup: rbacv1.GroupName,
		Name:     "Project Owner",
	}

	project := &storagev1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "core-project-test",
			Namespace: org.Status.Namespace.Name,
		},
		Spec: storagev1alpha1.ProjectSpec{
			Owners: []rbacv1.Subject{projectOwner},
		},
	}
	require.NoError(t, cl.Create(ctx, project))
	require.NoError(t, testutil.WaitUntilReady(ctx, cl, project))

	projectNs := &corev1.Namespace{}
	projectNs.Name = fmt.Sprintf("%s-%s", project.Namespace, project.Name)
	require.NoError(t, testutil.WaitUntilFound(ctx, cl, projectNs))

	// Make sure Role/RoleBinding for Organization Owner has been created in Project namespace.
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbacTemplate.Name,
			Namespace: projectNs.Name,
		},
	}
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbacTemplate.Name,
			Namespace: projectNs.Name,
		},
	}
	require.NoError(t, testutil.WaitUntilFound(ctx, cl, role))
	require.NoError(t, testutil.WaitUntilFound(ctx, cl, roleBinding))
	t.Log("Organization Owner has permission to create RoleBinding in Project namespace")
	cfg.Impersonate = rest.ImpersonationConfig{
		UserName: organizationOwner.Name,
	}
	ownerClient := testutil.NewRecordingClient(t, cfg, testScheme, testutil.CleanUpStrategy(cleanUpStrategy))
	rbacSubject := rbacv1.Subject{
		Kind:     rbacv1.UserKind,
		APIGroup: rbacv1.GroupName,
		Name:     "User1",
	}
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user1-rb",
			Namespace: projectNs.Name,
		},
		Subjects: []rbacv1.Subject{rbacSubject},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     rbacTemplate.Name,
		},
	}
	require.NoError(t, ownerClient.Create(ctx, rb))
	require.NoError(t, cl.WaitUntil(ctx, project, func() (done bool, err error) {
		if len(project.Status.Members) == 2 {
			assert.Contains(t, project.Status.Members, rbacSubject)
			// A RoleBinding for Organization Owner will also be created to grant Owner to have permission to create Role/RoleBindings in Project namespace.
			assert.Contains(t, project.Status.Members, organizationOwner)
			return true, nil
		}
		return false, nil
	}), "project didnt reconcile added member")
	require.NoError(t, testutil.DeleteAndWaitUntilNotFound(ctx, cl, rb))
	require.NoError(t, cl.WaitUntil(ctx, project, func() (done bool, err error) {
		return len(project.Status.Members) == 1, nil
	}), "project didnt reconcile removed member")

	require.NoError(t, cl.Delete(ctx, project))
	require.NoError(t, cl.WaitUntilNotFound(ctx, projectNs), "Project namespace has not been cleaned up")
}
