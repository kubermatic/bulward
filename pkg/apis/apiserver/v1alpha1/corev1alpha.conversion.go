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

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/apiserver-builder-alpha/pkg/builders"

	"github.com/kubermatic/bulward/pkg/apis/apiserver"
	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
)

func init() {
	utilruntime.Must(Corev1alpha1RegisterConversion(builders.Scheme))
}

func Corev1alpha1RegisterConversion(scheme *runtime.Scheme) error {
	if err := scheme.AddGeneratedConversionFunc((*apiserver.Organization)(nil), (*corev1alpha1.InternalOrganization)(nil), func(a, b interface{}, scope conversion.Scope) error {
		in := a.(*apiserver.Organization)
		out := b.(*corev1alpha1.InternalOrganization)
		out.ObjectMeta = in.ObjectMeta
		out.TypeMeta = in.TypeMeta
		for i := range out.ManagedFields {
			out.ManagedFields[i].APIVersion = corev1alpha1.GroupVersion.String()
		}
		return nil
	}); err != nil {
		return err
	}
	if err := scheme.AddGeneratedConversionFunc((*corev1alpha1.InternalOrganization)(nil), (*apiserver.Organization)(nil), func(a, b interface{}, scope conversion.Scope) error {
		in := a.(*corev1alpha1.InternalOrganization)
		out := b.(*apiserver.Organization)
		out.ObjectMeta = in.ObjectMeta
		out.TypeMeta = in.TypeMeta
		for i := range out.ManagedFields {
			out.ManagedFields[i].APIVersion = SchemeGroupVersion.String()
		}
		return nil
	}); err != nil {
		return err
	}
	if err := scheme.AddGeneratedConversionFunc((*apiserver.OrganizationList)(nil), (*corev1alpha1.InternalOrganizationList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		in := a.(*apiserver.OrganizationList)
		out := b.(*corev1alpha1.InternalOrganizationList)
		out.Items = nil
		for _, it := range in.Items {
			outIt := &corev1alpha1.InternalOrganization{}
			if err := scope.Convert(&it, outIt, scope.Flags()); err != nil {
				return err
			}
			out.Items = append(out.Items, *outIt)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := scheme.AddGeneratedConversionFunc((*corev1alpha1.InternalOrganizationList)(nil), (*apiserver.OrganizationList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		in := a.(*corev1alpha1.InternalOrganizationList)
		out := b.(*apiserver.OrganizationList)
		out.Items = nil
		for _, it := range in.Items {
			outIt := &apiserver.Organization{}
			if err := scope.Convert(&it, outIt, scope.Flags()); err != nil {
				return err
			}
			out.Items = append(out.Items, *outIt)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
