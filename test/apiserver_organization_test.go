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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/kubermatic/utils/pkg/testutil"

	apiserverv1alpha1 "github.com/kubermatic/bulward/pkg/apis/apiserver/v1alpha1"
	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
	storagev1alpha1 "github.com/kubermatic/bulward/pkg/apis/storage/v1alpha1"
)

func init() {
	utilruntime.Must(corev1.AddToScheme(testScheme))
	utilruntime.Must(corev1alpha1.AddToScheme(testScheme))
	utilruntime.Must(apiserverv1alpha1.AddToScheme(testScheme))
}

func TestAPIServerOrganization(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cfg, err := config.GetConfig()
	require.NoError(t, err)
	cl := testutil.NewRecordingClient(t, cfg, testScheme, testutil.CleanUpStrategy(cleanUpStrategy))
	require.NoError(t, err)
	dcl, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return
	}

	description := "I'm a little test organization from Berlin."
	owner := rbacv1.Subject{
		Kind:     "User",
		APIGroup: "rbac.authorization.k8s.io",
		Name:     "Owner1",
	}
	org := &apiserverv1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: storagev1alpha1.OrganizationSpec{
			Metadata: &storagev1alpha1.OrganizationMetadata{
				DisplayName: "test",
				Description: description,
			},
			Owners: []rbacv1.Subject{owner},
		},
	}

	gvr := apiserverv1alpha1.Resource("organizations").WithVersion("v1alpha1")
	wi, err := dcl.Resource(gvr).Watch(ctx, metav1.ListOptions{
		FieldSelector: "metadata.name=test",
	})
	require.NoError(t, err)
	t.Cleanup(wi.Stop)

	t.Log("create")
	require.NoError(t, cl.Create(ctx, org))
	for ev := range wi.ResultChan() {
		if ev.Type == watch.Added {
			obj := &apiserverv1alpha1.Organization{}
			require.NoError(t, scheme.Scheme.Convert(ev.Object, obj, nil))
			if assert.Equal(t, "test", obj.Name, "got non-test organization, meaning watch fieldSelector hasn't functioned properly") {
				assert.Equal(t, description, obj.Spec.Metadata.Description)
				t.Log("watch -- created")
				break
			}
		}
	}

	t.Log("get")
	require.NoError(t, cl.Get(ctx, types.NamespacedName{Name: org.Name}, org))
	assert.Equal(t, description, org.Spec.Metadata.Description, "description")

	t.Log("list")
	orgs := &apiserverv1alpha1.OrganizationList{}
	if assert.NoError(t, cl.List(ctx, orgs)) {
		if assert.Len(t, orgs.Items, 1) {
			assert.Equal(t, "test", orgs.Items[0].Name)
			assert.Equal(t, description, orgs.Items[0].Spec.Metadata.Description, "description")
		}
	}

modfor:
	for {
		select {
		case ev := <-wi.ResultChan():
			require.NoError(t, scheme.Scheme.Convert(ev.Object, org, nil))
			if assert.Equal(t, "test", org.Name, "got non-test organization, meaning watch fieldSelector hasn't functioned properly") {
				t.Log("watch -- modified")
			}
		case <-time.After(time.Second):
			break modfor
		}
	}

	assert.Equal(t, description, org.Spec.Metadata.Description, "description")
	assert.NotEqual(t, 0, org.Status.ObservedGeneration, "observed generation should be propagated")
	if assert.NotEmpty(t, org.Status.Namespace, "namespace is empty") {
		t.Log("org namespace: " + org.Status.Namespace.Name)
	}

	t.Log("update")
	require.NoError(t, testutil.TryUpdateUntil(ctx, cl, org, func() error {
		org.Labels = map[string]string{"aa": "bb"}
		return nil
	}))

	org = &apiserverv1alpha1.Organization{}
	require.NoError(t, cl.Get(ctx, types.NamespacedName{Name: "test"}, org))
	assert.Equal(t, "bb", org.Labels["aa"])

	t.Log("delete")
	assert.NoError(t, cl.Delete(ctx, org))

	for ev := range wi.ResultChan() {
		if ev.Type == watch.Deleted {
			obj := &apiserverv1alpha1.Organization{}
			require.NoError(t, scheme.Scheme.Convert(ev.Object, obj, nil))
			if assert.Equal(t, "test", obj.Name) {
				t.Log("watch -- deleted")
				break
			}
		}
	}
}
