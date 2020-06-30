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
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/apiserver-builder-alpha/pkg/builders"

	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
)

func init() {
	utilruntime.Must(Corev1alpha1RegisterConversion(builders.Scheme))
	utilruntime.Must(RegisterConversions(builders.Scheme))
	utilruntime.Must(RegisterDefaults(builders.Scheme))
}

func Corev1alpha1RegisterConversion(scheme *runtime.Scheme) error {
	if err := scheme.AddConversionFunc((*Organization)(nil), (*corev1alpha1.Organization)(nil), func(a, b interface{}, scope conversion.Scope) error {
		in := a.(*Organization)
		out := b.(*corev1alpha1.Organization)
		if err := Convert_apiserver_Organization_To_v1alpha1_Organization(in, out, scope); err != nil {
			return err
		}
		for i := range out.ManagedFields {
			out.ManagedFields[i].APIVersion = corev1alpha1.GroupVersion.String()
		}
		return nil
	}); err != nil {
		return err
	}
	if err := scheme.AddConversionFunc((*corev1alpha1.Organization)(nil), (*Organization)(nil), func(a, b interface{}, scope conversion.Scope) error {
		in := a.(*corev1alpha1.Organization)
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
	return nil
}
