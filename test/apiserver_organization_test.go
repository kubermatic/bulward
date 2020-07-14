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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	corev1 "k8s.io/client-go/tools/clientcmd/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	gvr = apiserverv1alpha1.Resource("organizations").WithVersion("v1alpha1")
)

func TestAPIServerOrganization(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cfg, err := config.GetConfig()
	require.NoError(t, err)
	cl := testutil.NewRecordingClient(t, cfg, testScheme, testutil.CleanUpStrategy(cleanUpStrategy))
	require.NoError(t, err)
	dcl, err := dynamic.NewForConfig(cfg)
	require.NoError(t, err)
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
		Spec: storagev1alpha1.OrganizationSpec{
			Metadata: &storagev1alpha1.OrganizationMetadata{
				DisplayName: "test",
				Description: description,
			},
			Owners: []rbacv1.Subject{owner},
		},
	}

	wi, err := dcl.Resource(gvr).Watch(ctx, metav1.ListOptions{
		FieldSelector: "metadata.name=test",
	})
	require.NoError(t, err)
	eventTracer := events.NewTracer(wi, events.IsObjectName("test"))
	t.Cleanup(eventTracer.TestCleanupFunc(t))

	t.Log("create")
	require.NoError(t, cl.Create(ctx, org))
	require.NoError(t, eventTracer.WaitUntil(ctx, events.IsType(watch.Added)))

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

	require.NoError(t, testutil.WaitUntilReady(ctx, cl, org))
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
	require.NoError(t, eventTracer.WaitUntil(ctx, events.IsType(watch.Modified)))

	org = &apiserverv1alpha1.Organization{}
	require.NoError(t, cl.Get(ctx, types.NamespacedName{Name: "test"}, org))
	assert.Equal(t, "bb", org.Labels["aa"])

	t.Log("delete")
	assert.NoError(t, cl.Delete(ctx, org))
	require.NoError(t, eventTracer.WaitUntil(ctx, events.IsType(watch.Deleted)))
}

type TestVisibleFilteringTestCase struct {
	Name                string
	Subject             rbacv1.Subject
	ImpersonationConfig rest.ImpersonationConfig
	Org                 *apiserverv1alpha1.Organization
	Client              *testutil.RecordingClient
	tracer              *events.Tracer
}

func TestVisibleFiltering(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cfg, err := config.GetConfig()
	require.NoError(t, err)
	cfg.UserAgent = t.Name()
	cl := testutil.NewRecordingClient(t, cfg, testScheme, testutil.CleanUpStrategy(cleanUpStrategy))
	t.Cleanup(cl.CleanUpFunc(ctx))
	dcl, err := dynamic.NewForConfig(cfg)
	require.NoError(t, err)
	wi, err := dcl.Resource(gvr).Watch(ctx, metav1.ListOptions{
		LabelSelector: "test-name=" + t.Name(),
	})
	require.NoError(t, err)
	globalEventTraced := events.NewTracer(wi)
	t.Cleanup(globalEventTraced.TestCleanupFunc(t))

	owner := rbacv1.Subject{
		Kind:     "User",
		APIGroup: "rbac.authorization.k8s.io",
		Name:     "kubernetes-admin",
	}
	testCase := []*TestVisibleFilteringTestCase{
		{
			Name: "user",
			Subject: rbacv1.Subject{
				Kind:     rbacv1.UserKind,
				APIGroup: rbacv1.GroupName,
				Name:     "user",
			},
			ImpersonationConfig: rest.ImpersonationConfig{
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
			ImpersonationConfig: rest.ImpersonationConfig{
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
			ImpersonationConfig: rest.ImpersonationConfig{
				UserName: "system:serviceaccount:default:default",
			},
		},
	}
	t.Log("creating orgs")
	for _, tc := range testCase {
		cfg, err := ctrl.GetConfig()
		require.NoError(t, err)
		cfg.Impersonate = tc.ImpersonationConfig
		cfg.UserAgent = t.Name() + "/" + tc.Name
		tc.Client = testutil.NewRecordingClient(t, cfg, testScheme, testutil.CleanUpStrategy(cleanUpStrategy))
		t.Cleanup(tc.Client.CleanUpFunc(ctx))
		dcl, err := dynamic.NewForConfig(cfg)
		require.NoError(t, err)
		wi, err := dcl.Resource(gvr).Watch(ctx, metav1.ListOptions{
			LabelSelector: "test-name=" + t.Name(),
		})
		require.NoError(t, err)
		tc.tracer = events.NewTracer(wi, events.IsObjectName("test-"+tc.Name))
		t.Cleanup(tc.tracer.TestCleanupFunc(t))
		tc.Org = &apiserverv1alpha1.Organization{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-" + tc.Name,
				Labels: map[string]string{
					"test-name": t.Name(),
				},
			},
			Spec: storagev1alpha1.OrganizationSpec{
				Metadata: &storagev1alpha1.OrganizationMetadata{
					DisplayName: "test",
					Description: "desc",
				},
				Owners: []rbacv1.Subject{owner},
			},
		}
		err = tc.Client.Create(ctx, tc.Org)
		require.True(t, errors.IsForbidden(err), "creating organization I'm not owner of should be forbidden")
		require.NoError(t, cl.Create(ctx, tc.Org))
		assert.NoError(t, globalEventTraced.WaitUntil(ctx, events.AllOf(
			events.IsType(watch.Added),
			events.IsObjectName(tc.Org.Name),
		)))
		require.NoError(t, testutil.WaitUntilReady(ctx, cl, tc.Org))
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
		assert.NoError(t, tc.tracer.WaitUntil(ctx, events.IsType(watch.Modified)))
	}

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

	for i, tc := range testCase {
		t.Run(tc.Name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(ctx)
			t.Cleanup(cancel)
			otherOrg := testCase[(i+1)%len(testCase)].Org

			orgs := &apiserverv1alpha1.OrganizationList{}
			require.NoError(t, tc.Client.List(ctx, orgs, client.MatchingLabels(tc.Org.Labels)))
			if assert.Len(t, orgs.Items, 1) {
				assert.Equal(t, tc.Org.Name, orgs.Items[0].Name)
			}

			org := &apiserverv1alpha1.Organization{}
			assert.True(t, errors.IsNotFound(tc.Client.Get(ctx, types.NamespacedName{Name: otherOrg.Name}, org)), "found forbidden org")
			require.NoError(t, tc.Client.Get(ctx, types.NamespacedName{Name: tc.Org.Name}, org), "get")
			err := testutil.TryUpdateUntil(ctx, tc.Client, org, func() error {
				org.Labels["aa"] = "bb"
				return nil
			})
			require.True(t, errors.IsForbidden(err), "update should be forbidden by non-owner, got %v", err)
			err = tc.Client.Delete(ctx, org)
			require.True(t, errors.IsForbidden(err), "delete should be forbidden by non-owners, got %v", err)
			require.NoError(t, cl.Delete(ctx, org))
			assert.NoError(t, tc.tracer.WaitUntil(ctx, events.IsType(watch.Deleted)))
		})
	}
}
