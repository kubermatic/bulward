/*
Copyright 2020 The Bulward Author.

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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
"k8s.io/apiserver/pkg/registry/generic"
"k8s.io/apiserver/pkg/registry/rest"

)

type organizationREST struct {
}

var _ rest.Storage = (*organizationREST)(nil)

var _ rest.Scoper = (*organizationREST)(nil)
var _ rest.Getter = (*organizationREST)(nil)
func NewOrganizationREST(getter generic.RESTOptionsGetter) rest.Storage {
	return &organizationREST{}
}

func (o *organizationREST) New() runtime.Object {
	return &Organization{}
}

func (o *organizationREST) NamespaceScoped() bool {
	return false
}

func (o *organizationREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return &Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-org",
		},
	}, nil
}
