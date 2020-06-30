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

package apiserver

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
)

func chainConversion(scheme *runtime.Scheme, initObj runtime.Object, objs ...runtime.Object) (runtime.Object, error) {
	objs = append([]runtime.Object{initObj}, objs...)
	for i := 0; i+1 < len(objs); i++ {
		if err := scheme.Convert(objs[i], objs[i+1], nil); err != nil {
			return nil, err
		}
	}
	return objs[len(objs)-1], nil
}

func ConvertToV1Alpha1Unstructured(organization *Organization, scheme *runtime.Scheme) (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	if _, err := chainConversion(scheme, organization, &corev1alpha1.InternalOrganization{}, u); err != nil {
		return nil, err
	}
	return u, nil
}

func ConvertFromV1Alpha1Unstructured(internalOrgv1alpha1 *unstructured.Unstructured, scheme *runtime.Scheme) (*Organization, error) {
	gvk, err := apiutil.GVKForObject(internalOrgv1alpha1, scheme)
	if err != nil {
		return nil, err
	}
	expectedGVK := corev1alpha1.GroupVersion.WithKind("InternalOrganization")
	if gvk != expectedGVK {
		return nil, fmt.Errorf("wrong GVK, expected %v, found %v", expectedGVK, gvk)
	}
	org := &Organization{}
	if _, err := chainConversion(scheme, internalOrgv1alpha1, &corev1alpha1.InternalOrganization{}, org); err != nil {
		return nil, err
	}
	return org, nil
}

func ConvertFromV1Alpha1UnstructuredList(internalOrgv1alpha1 *unstructured.UnstructuredList, scheme *runtime.Scheme) (*OrganizationList, error) {
	sol := &OrganizationList{}
	for _, it := range internalOrgv1alpha1.Items {
		org, err := ConvertFromV1Alpha1Unstructured(&it, scheme)
		if err != nil {
			return nil, err
		}
		sol.Items = append(sol.Items, *org)
	}
	return sol, nil
}

func ConvertToV1Alpha1UnstructuredList(organizations *OrganizationList, scheme *runtime.Scheme) (*unstructured.UnstructuredList, error) {
	sol := &unstructured.UnstructuredList{}
	for _, it := range organizations.Items {
		org, err := ConvertToV1Alpha1Unstructured(&it, scheme)
		if err != nil {
			return nil, err
		}
		sol.Items = append(sol.Items, *org)
	}
	return sol, nil
}