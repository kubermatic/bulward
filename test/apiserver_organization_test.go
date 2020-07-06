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
	"k8s.io/client-go/rest"
	corev1 "k8s.io/client-go/tools/clientcmd/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/kubermatic/utils/pkg/testutil"

	apiserverv1alpha1 "github.com/kubermatic/bulward/pkg/apis/apiserver/v1alpha1"
	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
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
	cl := testutil.NewRecordingClient(t, cfg, testScheme, testutil.CleanupOnSuccess)
	require.NoError(t, err)
	dcl, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return
	}

	description := "I'm a little test organization from Berlin."
	owner := rbacv1.Subject{
		Kind:     "User",
		APIGroup: "rbac.authorization.k8s.io",
		Name:     "kubernetes-admin",
	}
	org := &apiserverv1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: corev1alpha1.OrganizationSpec{
			Metadata: &corev1alpha1.OrganizationMetadata{
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

func TestVisibleFiltering(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cfg, err := config.GetConfig()
	require.NoError(t, err)
	cfg.UserAgent = t.Name()
	cl := testutil.NewRecordingClient(t, cfg, testScheme, testutil.CleanUpStrategy(cleanUpStragety))
	t.Cleanup(cl.CleanUpFunc(ctx))

	owner := rbacv1.Subject{
		Kind:     "User",
		APIGroup: "rbac.authorization.k8s.io",
		Name:     "kubernetes-admin",
	}
	testCase := []*struct {
		Name    string
		Subject rbacv1.Subject
		Imp     rest.ImpersonationConfig
		Org     *apiserverv1alpha1.Organization
	}{
		{
			Name: "user",
			Subject: rbacv1.Subject{
				Kind:     rbacv1.UserKind,
				APIGroup: rbacv1.GroupName,
				Name:     "user",
			},
			Imp: rest.ImpersonationConfig{
				UserName: "user",
			},
		},
		{
			Name: "group",
			Subject: rbacv1.Subject{
				Kind:     rbacv1.GroupKind,
				APIGroup: rbacv1.GroupName,
				Name:     "group",
			},
			Imp: rest.ImpersonationConfig{
				UserName: "lala",
				// without "system:authenticated" things are breaking...cannot RESTMapper doesn't function
				Groups: []string{"system:authenticated", "group"},
			},
		},
		{
			Name: "sa",
			Subject: rbacv1.Subject{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      "default",
				Namespace: "default",
			},
			Imp: rest.ImpersonationConfig{
				UserName: "system:serviceaccount:default:default",
			},
		},
	}

	t.Log("creating orgs")
	for _, tc := range testCase {
		tc.Org = &apiserverv1alpha1.Organization{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-" + tc.Name,
				Labels: map[string]string{
					"test-name": t.Name(),
				},
			},
			Spec: corev1alpha1.OrganizationSpec{
				Metadata: &corev1alpha1.OrganizationMetadata{
					DisplayName: "test",
					Description: "desc",
				},
				Owners: []rbacv1.Subject{owner},
			},
		}
		require.NoError(t, cl.Create(ctx, tc.Org))
	}

	t.Log("waiting for orgs ready")
	for _, tc := range testCase {
		require.NoError(t, testutil.WaitUntilReady(ctx, cl, tc.Org))
	}

	t.Log("creating rolebindings in the orgs")
	for _, tc := range testCase {
		rb := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rb",
				Namespace: tc.Org.Status.Namespace.Name,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     "test",
			},
			Subjects: []rbacv1.Subject{tc.Subject},
		}
		require.NoError(t, cl.Create(ctx, rb))
	}

	t.Log("creating necessary cluster roles/rolebinding")
	crole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "bulward:test-visible-filtering",
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{rbacv1.VerbAll},
				APIGroups: []string{apiserverv1alpha1.SchemeGroupVersion.Group},
				Resources: []string{rbacv1.ResourceAll},
			},
		},
	}
	crolebinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "bulward:test-visible-filtering",
		},
		RoleRef: rbacv1.RoleRef{
			Name: crole.Name,
			Kind: "ClusterRole",
		},
	}
	for _, tc := range testCase {
		crolebinding.Subjects = append(crolebinding.Subjects, tc.Subject)
	}
	require.NoError(t, cl.EnsureCreated(ctx, crole))
	require.NoError(t, cl.Create(ctx, crolebinding))

	t.Log("waiting for orgs member status updates")
	for _, tc := range testCase {
		require.NoError(t, cl.WaitUntil(ctx, tc.Org, func() (done bool, err error) {
			for _, member := range tc.Org.Status.Members {
				if member == tc.Subject {
					return true, nil
				}
			}
			return false, nil
		}))
	}

	for _, tc := range testCase {
		t.Run(tc.Name, func(t *testing.T) {
			cfg, err := ctrl.GetConfig()
			require.NoError(t, err)
			cfg.Impersonate = tc.Imp
			impCl, err := client.New(cfg, client.Options{
				Scheme: testScheme,
			})
			require.NoError(t, err)
			orgs := &apiserverv1alpha1.OrganizationList{}
			require.NoError(t, impCl.List(ctx, orgs, client.MatchingLabels(tc.Org.Labels)))
			if assert.Len(t, orgs.Items, 1) {
				assert.Equal(t, tc.Org.Name, orgs.Items[0].Name)
			}
		})
	}
}
