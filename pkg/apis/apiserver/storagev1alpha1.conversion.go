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
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	storagev1alpha1 "github.com/kubermatic/bulward/pkg/apis/storage/v1alpha1"
)

func init() {
	localSchemeBuilder.Register(Corev1alpha1RegisterConversion)
}

func Corev1alpha1RegisterConversion(scheme *runtime.Scheme) error {
	if err := scheme.AddConversionFunc((*Organization)(nil), (*storagev1alpha1.Organization)(nil), func(a, b interface{}, scope conversion.Scope) error {
		in := a.(*Organization)
		out := b.(*storagev1alpha1.Organization)
		if err := Convert_apiserver_Organization_To_v1alpha1_Organization(in, out, scope); err != nil {
			return err
		}
		for i := range out.ManagedFields {
			out.ManagedFields[i].APIVersion = storagev1alpha1.GroupVersion.String()
		}
		return nil
	}); err != nil {
		return err
	}
	if err := scheme.AddConversionFunc((*storagev1alpha1.Organization)(nil), (*Organization)(nil), func(a, b interface{}, scope conversion.Scope) error {
		in := a.(*storagev1alpha1.Organization)
		out := b.(*Organization)
		if err := Convert_v1alpha1_Organization_To_apiserver_Organization(in, out, scope); err != nil {
			return err
		}
		for i := range out.ManagedFields {
			out.ManagedFields[i].APIVersion = schema.GroupVersion{
				Group:   SchemeGroupVersion.Group,
				Version: "v1alpha1",
			}.String()
		}
		return nil
	}); err != nil {
		return err
	}
	if err := scheme.AddConversionFunc((*Project)(nil), (*storagev1alpha1.Project)(nil), func(a, b interface{}, scope conversion.Scope) error {
		in := a.(*Project)
		out := b.(*storagev1alpha1.Project)
		if err := Convert_apiserver_Project_To_v1alpha1_Project(in, out, scope); err != nil {
			return err
		}
		for i := range out.ManagedFields {
			out.ManagedFields[i].APIVersion = storagev1alpha1.GroupVersion.String()
		}
		return nil
	}); err != nil {
		return err
	}
	if err := scheme.AddConversionFunc((*storagev1alpha1.Project)(nil), (*Project)(nil), func(a, b interface{}, scope conversion.Scope) error {
		in := a.(*storagev1alpha1.Project)
		out := b.(*Project)
		if err := Convert_v1alpha1_Project_To_apiserver_Project(in, out, scope); err != nil {
			return err
		}
		for i := range out.ManagedFields {
			out.ManagedFields[i].APIVersion = schema.GroupVersion{
				Group:   SchemeGroupVersion.Group,
				Version: "v1alpha1",
			}.String()
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
