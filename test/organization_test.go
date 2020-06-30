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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	apiserverv1alpha1 "github.com/kubermatic/bulward/pkg/apis/apiserver/v1alpha1"
	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
)

func init() {
	utilruntime.Must(corev1.AddToScheme(scheme.Scheme))
	utilruntime.Must(corev1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(apiserverv1alpha1.AddToScheme(scheme.Scheme))
}

func TestIntegration(t *testing.T) {
	org := &apiserverv1alpha1.Organization{}
	org.Name = "test"
	cfg, err := config.GetConfig()
	require.NoError(t, err)
	cl, err := client.New(cfg, client.Options{
		Scheme: scheme.Scheme,
	})
	require.NoError(t, err)
	dcl, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	gvr := apiserverv1alpha1.Resource("organizations").WithVersion("v1alpha1")
	wi, err := dcl.Resource(gvr).Watch(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	t.Cleanup(wi.Stop)

	t.Log("create")
	require.NoError(t, cl.Create(ctx, org))
	for ev := range wi.ResultChan() {
		if ev.Type == watch.Added {
			obj := &apiserverv1alpha1.Organization{}
			require.NoError(t, scheme.Scheme.Convert(ev.Object, obj, nil))
			if obj.Name == "test" {
				t.Log("watch -- created")
				break
			}
		}
	}

	t.Log("get")
	require.NoError(t, cl.Get(ctx, types.NamespacedName{Name: org.Name}, org))

	t.Log("list")
	orgs := &apiserverv1alpha1.OrganizationList{}
	if assert.NoError(t, cl.List(ctx, orgs)) {
		if assert.Len(t, orgs.Items, 1) {
			assert.Equal(t, "test", orgs.Items[0].Name)
		}
	}

modfor:
	for {
		select {
		case ev := <-wi.ResultChan():
			require.NoError(t, scheme.Scheme.Convert(ev.Object, org, nil))
			if org.Name == "test" {
				t.Log("watch -- modified")
			}
		case <-time.After(time.Second):
			break modfor
		}
	}

	t.Log("update")
	org.Labels = map[string]string{"aa": "bb"}
	assert.NoError(t, cl.Update(ctx, org))

	org = &apiserverv1alpha1.Organization{}
	org.Name = "test"
	require.NoError(t, cl.Get(ctx, types.NamespacedName{Name: org.Name}, org))
	assert.Equal(t, "bb", org.Labels["aa"])

	t.Log("delete")
	assert.NoError(t, cl.Delete(ctx, org))

	for ev := range wi.ResultChan() {
		if ev.Type == watch.Deleted {
			obj := &apiserverv1alpha1.Organization{}
			require.NoError(t, scheme.Scheme.Convert(ev.Object, obj, nil))
			if obj.Name == "test" {
				t.Log("watch -- deleted")
				break
			}
		}
	}
}
