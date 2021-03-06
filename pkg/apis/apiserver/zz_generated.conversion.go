// +build !ignore_autogenerated

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

// Code generated by conversion-gen. DO NOT EDIT.

package apiserver

import (
	unsafe "unsafe"

	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"

	v1alpha1 "k8c.io/bulward/pkg/apis/storage/v1alpha1"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*Organization)(nil), (*v1alpha1.Organization)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_apiserver_Organization_To_v1alpha1_Organization(a.(*Organization), b.(*v1alpha1.Organization), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha1.Organization)(nil), (*Organization)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_Organization_To_apiserver_Organization(a.(*v1alpha1.Organization), b.(*Organization), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*OrganizationList)(nil), (*v1alpha1.OrganizationList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_apiserver_OrganizationList_To_v1alpha1_OrganizationList(a.(*OrganizationList), b.(*v1alpha1.OrganizationList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha1.OrganizationList)(nil), (*OrganizationList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_OrganizationList_To_apiserver_OrganizationList(a.(*v1alpha1.OrganizationList), b.(*OrganizationList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*Project)(nil), (*v1alpha1.Project)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_apiserver_Project_To_v1alpha1_Project(a.(*Project), b.(*v1alpha1.Project), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha1.Project)(nil), (*Project)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_Project_To_apiserver_Project(a.(*v1alpha1.Project), b.(*Project), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*ProjectList)(nil), (*v1alpha1.ProjectList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_apiserver_ProjectList_To_v1alpha1_ProjectList(a.(*ProjectList), b.(*v1alpha1.ProjectList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v1alpha1.ProjectList)(nil), (*ProjectList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_ProjectList_To_apiserver_ProjectList(a.(*v1alpha1.ProjectList), b.(*ProjectList), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_apiserver_Organization_To_v1alpha1_Organization(in *Organization, out *v1alpha1.Organization, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	out.Status = in.Status
	return nil
}

// Convert_apiserver_Organization_To_v1alpha1_Organization is an autogenerated conversion function.
func Convert_apiserver_Organization_To_v1alpha1_Organization(in *Organization, out *v1alpha1.Organization, s conversion.Scope) error {
	return autoConvert_apiserver_Organization_To_v1alpha1_Organization(in, out, s)
}

func autoConvert_v1alpha1_Organization_To_apiserver_Organization(in *v1alpha1.Organization, out *Organization, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	out.Status = in.Status
	return nil
}

// Convert_v1alpha1_Organization_To_apiserver_Organization is an autogenerated conversion function.
func Convert_v1alpha1_Organization_To_apiserver_Organization(in *v1alpha1.Organization, out *Organization, s conversion.Scope) error {
	return autoConvert_v1alpha1_Organization_To_apiserver_Organization(in, out, s)
}

func autoConvert_apiserver_OrganizationList_To_v1alpha1_OrganizationList(in *OrganizationList, out *v1alpha1.OrganizationList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]v1alpha1.Organization)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_apiserver_OrganizationList_To_v1alpha1_OrganizationList is an autogenerated conversion function.
func Convert_apiserver_OrganizationList_To_v1alpha1_OrganizationList(in *OrganizationList, out *v1alpha1.OrganizationList, s conversion.Scope) error {
	return autoConvert_apiserver_OrganizationList_To_v1alpha1_OrganizationList(in, out, s)
}

func autoConvert_v1alpha1_OrganizationList_To_apiserver_OrganizationList(in *v1alpha1.OrganizationList, out *OrganizationList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]Organization)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v1alpha1_OrganizationList_To_apiserver_OrganizationList is an autogenerated conversion function.
func Convert_v1alpha1_OrganizationList_To_apiserver_OrganizationList(in *v1alpha1.OrganizationList, out *OrganizationList, s conversion.Scope) error {
	return autoConvert_v1alpha1_OrganizationList_To_apiserver_OrganizationList(in, out, s)
}

func autoConvert_apiserver_Project_To_v1alpha1_Project(in *Project, out *v1alpha1.Project, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	out.Status = in.Status
	return nil
}

// Convert_apiserver_Project_To_v1alpha1_Project is an autogenerated conversion function.
func Convert_apiserver_Project_To_v1alpha1_Project(in *Project, out *v1alpha1.Project, s conversion.Scope) error {
	return autoConvert_apiserver_Project_To_v1alpha1_Project(in, out, s)
}

func autoConvert_v1alpha1_Project_To_apiserver_Project(in *v1alpha1.Project, out *Project, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	out.Status = in.Status
	return nil
}

// Convert_v1alpha1_Project_To_apiserver_Project is an autogenerated conversion function.
func Convert_v1alpha1_Project_To_apiserver_Project(in *v1alpha1.Project, out *Project, s conversion.Scope) error {
	return autoConvert_v1alpha1_Project_To_apiserver_Project(in, out, s)
}

func autoConvert_apiserver_ProjectList_To_v1alpha1_ProjectList(in *ProjectList, out *v1alpha1.ProjectList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]v1alpha1.Project)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_apiserver_ProjectList_To_v1alpha1_ProjectList is an autogenerated conversion function.
func Convert_apiserver_ProjectList_To_v1alpha1_ProjectList(in *ProjectList, out *v1alpha1.ProjectList, s conversion.Scope) error {
	return autoConvert_apiserver_ProjectList_To_v1alpha1_ProjectList(in, out, s)
}

func autoConvert_v1alpha1_ProjectList_To_apiserver_ProjectList(in *v1alpha1.ProjectList, out *ProjectList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]Project)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v1alpha1_ProjectList_To_apiserver_ProjectList is an autogenerated conversion function.
func Convert_v1alpha1_ProjectList_To_apiserver_ProjectList(in *v1alpha1.ProjectList, out *ProjectList, s conversion.Scope) error {
	return autoConvert_v1alpha1_ProjectList_To_apiserver_ProjectList(in, out, s)
}
