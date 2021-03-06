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

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	storagev1alpha1 "k8c.io/bulward/pkg/apis/storage/v1alpha1"
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

func ConvertToUnstructuredStorageV1Alpha1Organization(organization *Organization, scheme *runtime.Scheme) (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	if _, err := chainConversion(scheme, organization, &storagev1alpha1.Organization{}, u); err != nil {
		return nil, err
	}
	return u, nil
}

func ConvertFromUnstructuredStorageV1Alpha1Organization(internalOrgv1alpha1 *unstructured.Unstructured, scheme *runtime.Scheme) (*Organization, error) {
	gvk, err := apiutil.GVKForObject(internalOrgv1alpha1, scheme)
	if err != nil {
		return nil, err
	}
	expectedGVK := storagev1alpha1.SchemeGroupVersion.WithKind("Organization")
	if gvk != expectedGVK {
		return nil, fmt.Errorf("wrong GVK, expected %v, found %v", expectedGVK, gvk)
	}
	org := &Organization{}
	if _, err := chainConversion(scheme, internalOrgv1alpha1, &storagev1alpha1.Organization{}, org); err != nil {
		return nil, err
	}
	return org, nil
}

func ConvertToUnstructuredStorageV1Alpha1OrganizationList(organizations *OrganizationList, scheme *runtime.Scheme) (*unstructured.UnstructuredList, error) {
	accesssor, err := meta.ListAccessor(organizations)
	if err != nil {
		return nil, err
	}
	sol := &unstructured.UnstructuredList{}
	for _, it := range organizations.Items {
		org, err := ConvertToUnstructuredStorageV1Alpha1Organization(&it, scheme)
		if err != nil {
			return nil, err
		}
		sol.Items = append(sol.Items, *org)
	}
	sol.SetResourceVersion(accesssor.GetResourceVersion())
	sol.SetContinue(accesssor.GetContinue())
	sol.SetRemainingItemCount(accesssor.GetRemainingItemCount())
	return sol, nil
}

func ConvertFromUnstructuredStorageV1Alpha1OrganizationList(internalOrgv1alpha1 *unstructured.UnstructuredList, scheme *runtime.Scheme) (*OrganizationList, error) {
	accesssor, err := meta.ListAccessor(internalOrgv1alpha1)
	if err != nil {
		return nil, err
	}
	sol := &OrganizationList{}
	for _, it := range internalOrgv1alpha1.Items {
		org, err := ConvertFromUnstructuredStorageV1Alpha1Organization(&it, scheme)
		if err != nil {
			return nil, err
		}
		sol.Items = append(sol.Items, *org)
	}
	sol.SetResourceVersion(accesssor.GetResourceVersion())
	sol.SetContinue(accesssor.GetContinue())
	sol.SetRemainingItemCount(accesssor.GetRemainingItemCount())
	return sol, nil
}
