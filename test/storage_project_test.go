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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/kubermatic/utils/pkg/testutil"

	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
	storagev1alpha1 "github.com/kubermatic/bulward/pkg/apis/storage/v1alpha1"
)

func init() {
	utilruntime.Must(corev1alpha1.AddToScheme(testScheme))
	utilruntime.Must(clientgoscheme.AddToScheme(testScheme))
	utilruntime.Must(storagev1alpha1.AddToScheme(testScheme))
}

func TestCoreProject(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cfg, err := controllerruntime.GetConfig()
	require.NoError(t, err)
	cl := testutil.NewRecordingClient(t, cfg, testScheme, testutil.CleanUpStrategy(cleanUpStrategy))
	t.Cleanup(cl.CleanUpFunc(ctx))

	ns := &corev1.Namespace{}
	ns.Name = "core-organization-test"

	owner := rbacv1.Subject{
		Kind:     "User",
		APIGroup: "rbac.authorization.k8s.io",
		Name:     "Owner1",
	}

	project := &storagev1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "core-project-test",
			Namespace: "core-organization-test",
		},
		Spec: storagev1alpha1.ProjectSpec{
			Owners: []rbacv1.Subject{owner},
		},
	}
	require.NoError(t, testutil.DeleteAndWaitUntilNotFound(ctx, cl, project))
	require.NoError(t, cl.Create(ctx, ns))
	require.NoError(t, cl.Create(ctx, project))
	require.NoError(t, testutil.WaitUntilReady(ctx, cl, project))

	projectNs := &corev1.Namespace{}
	projectNs.Name = fmt.Sprintf("%s-%s", project.Namespace, project.Name)
	require.NoError(t, testutil.WaitUntilFound(ctx, cl, projectNs))

	rbacSubject := rbacv1.Subject{
		Kind:     "User",
		APIGroup: "rbac.authorization.k8s.io",
		Name:     "User1",
	}
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user1-rb",
			Namespace: projectNs.Name,
		},
		Subjects: []rbacv1.Subject{rbacSubject},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "role",
		},
	}
	require.NoError(t, cl.Create(ctx, rb))
	require.NoError(t, cl.WaitUntil(ctx, project, func() (done bool, err error) {
		if len(project.Status.Members) == 1 {
			assert.Contains(t, project.Status.Members, rbacSubject)
			return true, nil
		}
		return false, nil
	}), "project didnt reconcile added member")
	require.NoError(t, testutil.DeleteAndWaitUntilNotFound(ctx, cl, rb))
	require.NoError(t, cl.WaitUntil(ctx, project, func() (done bool, err error) {
		return len(project.Status.Members) == 0, nil
	}), "project didnt reconcile removed member")
}
