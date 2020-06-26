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
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

type OrganizationREST struct {
	client  client.Client
	dynamic dynamic.Interface
	mapper  meta.RESTMapper
	scheme  *runtime.Scheme
}

var _ inject.Client = (*OrganizationREST)(nil)
var _ inject.Mapper = (*OrganizationREST)(nil)
var _ inject.Scheme = (*OrganizationREST)(nil)

func (o *OrganizationREST) InjectMapper(mapper meta.RESTMapper) error {
	if o.mapper != nil {
		return fmt.Errorf("mapper already injected")
	}
	o.mapper = mapper
	return nil
}

func (o *OrganizationREST) InjectClient(c client.Client) error {
	if o.client != nil {
		return fmt.Errorf("client already injected")
	}
	o.client = c
	return nil
}

func (o *OrganizationREST) InjectDynamicClient(dynamic dynamic.Interface) error {
	if o.dynamic != nil {
		return fmt.Errorf("dynamic already injected")
	}
	o.dynamic = dynamic
	return nil
}

func (o *OrganizationREST) InjectScheme(scheme *runtime.Scheme) error {
	if o.scheme != nil {
		return fmt.Errorf("scheme already injected")
	}
	o.scheme = scheme
	return nil
}

var _ rest.Storage = (*OrganizationREST)(nil)
var _ rest.Scoper = (*OrganizationREST)(nil)
var _ rest.Getter = (*OrganizationREST)(nil)
var _ rest.Lister = (*OrganizationREST)(nil)

var OrganizationRESTSingleton = &OrganizationREST{}

func NewOrganizationREST(_ generic.RESTOptionsGetter) rest.Storage {
	return OrganizationRESTSingleton
}

var fakeOrg = &Organization{
	ObjectMeta: metav1.ObjectMeta{
		Name: "fake-org",
	},
}

func (o *OrganizationREST) New() runtime.Object {
	return &Organization{}
}

func (o *OrganizationREST) NamespaceScoped() bool {
	return false
}

func (o *OrganizationREST) NewList() runtime.Object {
	return &OrganizationList{}
}

func (o *OrganizationREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return fakeOrg, nil
}

func (o *OrganizationREST) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	// TODO: Replace with a read implementation
	// small test whether the client is properly injected
	s := &v1.NamespaceList{}
	if err := o.client.List(ctx, s); err != nil {
		return nil, err
	}
	return &OrganizationList{Items: []Organization{*fakeOrg}}, nil
}

func (o *OrganizationREST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return rest.NewDefaultTableConvertor(Resource("organizations")).ConvertToTable(ctx, object, tableOptions)
}
