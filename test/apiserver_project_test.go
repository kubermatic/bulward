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
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	corev1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/kubermatic/utils/pkg/testutil"

	apiserverv1alpha1 "github.com/kubermatic/bulward/pkg/apis/apiserver/v1alpha1"
	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
	storagev1alpha1 "github.com/kubermatic/bulward/pkg/apis/storage/v1alpha1"
	"github.com/kubermatic/bulward/test/events"
)

func init() {
	utilruntime.Must(corev1.AddToScheme(testScheme))
	utilruntime.Must(corev1alpha1.AddToScheme(testScheme))
	utilruntime.Must(apiserverv1alpha1.AddToScheme(testScheme))
}

var (
	projectGvr = apiserverv1alpha1.Resource("projects").WithVersion("v1alpha1")
)

func TestAPIServerProject(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cfg, err := config.GetConfig()
	require.NoError(t, err)
	cl := testutil.NewRecordingClient(t, cfg, testScheme, testutil.CleanUpStrategy(cleanUpStrategy))
	require.NoError(t, err)
	dcl, err := dynamic.NewForConfig(cfg)
	require.NoError(t, err)
	ns := &v1.Namespace{}
	ns.Name = "test-org"
	require.NoError(t, cl.Create(ctx, ns))

	owner := rbacv1.Subject{
		Kind:     rbacv1.UserKind,
		APIGroup: rbacv1.GroupName,
		Name:     "kubernetes-admin",
	}
	project := &apiserverv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns.Name,
		},
		Spec: storagev1alpha1.ProjectSpec{
			Owners: []rbacv1.Subject{owner},
		},
	}

	wi, err := dcl.Resource(projectGvr).Watch(ctx, metav1.ListOptions{
		FieldSelector: "metadata.name=test",
	})
	require.NoError(t, err)
	eventTracer := events.NewTracer(wi, events.IsObjectName("test"))
	t.Cleanup(eventTracer.TestCleanupFunc(t))

	t.Log("create")
	require.NoError(t, cl.Create(ctx, project))
	require.NoError(t, eventTracer.WaitUntil(ctx, events.IsType(watch.Added)))

	t.Log("get")
	require.NoError(t, cl.Get(ctx, types.NamespacedName{Name: project.Name, Namespace: ns.Name}, project))
	assert.Contains(t, project.Spec.Owners, owner, "owner")

	t.Log("list")
	projects := &apiserverv1alpha1.ProjectList{}
	if assert.NoError(t, cl.List(ctx, projects)) {
		if assert.Len(t, projects.Items, 1) {
			assert.Equal(t, "test", projects.Items[0].Name)
			assert.Contains(t, project.Spec.Owners, owner, "owner")
		}
	}

	require.NoError(t, testutil.WaitUntilReady(ctx, cl, project))
	assert.Contains(t, project.Spec.Owners, owner, "owner")
	assert.NotEqual(t, 0, project.Status.ObservedGeneration, "observed generation should be propagated")
	if assert.NotEmpty(t, project.Status.Namespace, "namespace is empty") {
		t.Log("project namespace: " + project.Status.Namespace.Name)
	}

	t.Log("update")
	require.NoError(t, testutil.TryUpdateUntil(ctx, cl, project, func() error {
		project.Labels = map[string]string{"aa": "bb"}
		return nil
	}))
	require.NoError(t, eventTracer.WaitUntil(ctx, events.IsType(watch.Modified)))

	project = &apiserverv1alpha1.Project{}
	require.NoError(t, cl.Get(ctx, types.NamespacedName{Name: "test"}, project))
	assert.Equal(t, "bb", project.Labels["aa"])

	t.Log("delete")
	assert.NoError(t, cl.Delete(ctx, project))
	require.NoError(t, eventTracer.WaitUntil(ctx, events.IsType(watch.Deleted)))
}
