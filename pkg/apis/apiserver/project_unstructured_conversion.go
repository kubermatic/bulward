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

	storagev1alpha1 "github.com/kubermatic/bulward/pkg/apis/storage/v1alpha1"
)

func ConvertToUnstructuredStorageV1Alpha1Project(project *Project, scheme *runtime.Scheme) (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	if _, err := chainConversion(scheme, project, &storagev1alpha1.Project{}, u); err != nil {
		return nil, err
	}
	return u, nil
}

func ConvertFromUnstructuredStorageV1Alpha1Project(internalProjectv1alpha1 *unstructured.Unstructured, scheme *runtime.Scheme) (*Project, error) {
	gvk, err := apiutil.GVKForObject(internalProjectv1alpha1, scheme)
	if err != nil {
		return nil, err
	}
	expectedGVK := storagev1alpha1.GroupVersion.WithKind("Project")
	if gvk != expectedGVK {
		return nil, fmt.Errorf("wrong GVK, expected %v, found %v", expectedGVK, gvk)
	}
	project := &Project{}
	if _, err := chainConversion(scheme, internalProjectv1alpha1, &storagev1alpha1.Project{}, project); err != nil {
		return nil, err
	}
	return project, nil
}

func ConvertToUnstructuredStorageV1Alpha1ProjectList(projects *ProjectList, scheme *runtime.Scheme) (*unstructured.UnstructuredList, error) {
	accesssor, err := meta.ListAccessor(projects)
	if err != nil {
		return nil, err
	}
	spl := &unstructured.UnstructuredList{}
	for _, it := range projects.Items {
		project, err := ConvertToUnstructuredStorageV1Alpha1Project(&it, scheme)
		if err != nil {
			return nil, err
		}
		spl.Items = append(spl.Items, *project)
	}
	spl.SetSelfLink(fmt.Sprintf("/apis/%s/%s/%s", storagev1alpha1.GroupVersion.Group, storagev1alpha1.GroupVersion.Version, internalProjectResource))
	spl.SetResourceVersion(accesssor.GetResourceVersion())
	spl.SetContinue(accesssor.GetContinue())
	spl.SetRemainingItemCount(accesssor.GetRemainingItemCount())
	return spl, nil
}

func ConvertFromUnstructuredStorageV1Alpha1ProjectList(internalProjectv1alpha1 *unstructured.UnstructuredList, scheme *runtime.Scheme) (*ProjectList, error) {
	accesssor, err := meta.ListAccessor(internalProjectv1alpha1)
	if err != nil {
		return nil, err
	}
	spl := &ProjectList{}
	for _, it := range internalProjectv1alpha1.Items {
		project, err := ConvertFromUnstructuredStorageV1Alpha1Project(&it, scheme)
		if err != nil {
			return nil, err
		}
		spl.Items = append(spl.Items, *project)
	}
	spl.SetSelfLink(fmt.Sprintf("/apis/%s/%s/%s", SchemeGroupVersion.Group, "v1alpha1", externalProjectResource))
	spl.SetResourceVersion(accesssor.GetResourceVersion())
	spl.SetContinue(accesssor.GetContinue())
	spl.SetRemainingItemCount(accesssor.GetRemainingItemCount())
	return spl, nil
}
