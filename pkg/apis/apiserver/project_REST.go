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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/filters"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	storagev1alpha1 "k8c.io/bulward/pkg/apis/storage/v1alpha1"
)

const (
	internalProjectResource = "projects"
	externalProjectResource = "projects"
)

// +k8s:deepcopy-gen=false
type ProjectREST struct {
	client    client.Client
	dynamicRI dynamic.NamespaceableResourceInterface
	mapper    meta.RESTMapper
	scheme    *runtime.Scheme
}

var ProjectRESTSingleton = &ProjectREST{}

func NewProjectREST(_ generic.RESTOptionsGetter) rest.Storage {
	return ProjectRESTSingleton
}

var _ inject.Client = (*ProjectREST)(nil)
var _ inject.Mapper = (*ProjectREST)(nil)
var _ inject.Scheme = (*ProjectREST)(nil)

func (p *ProjectREST) InjectMapper(mapper meta.RESTMapper) error {
	if p.mapper != nil {
		return fmt.Errorf("mapper already injected")
	}
	p.mapper = mapper
	return nil
}
func (p *ProjectREST) InjectClient(c client.Client) error {
	if p.client != nil {
		return fmt.Errorf("client already injected")
	}
	p.client = c
	return nil
}

func (p *ProjectREST) InjectDynamicClient(dynamic dynamic.Interface) error {
	if p.dynamicRI != nil {
		return fmt.Errorf("dynamicRI already injected")
	}
	p.dynamicRI = dynamic.Resource(storagev1alpha1.SchemeGroupVersion.WithResource(internalProjectResource))
	return nil
}

func (p *ProjectREST) InjectScheme(scheme *runtime.Scheme) error {
	if p.scheme != nil {
		return fmt.Errorf("scheme already injected")
	}
	p.scheme = scheme
	return nil
}

var _ rest.Storage = (*ProjectREST)(nil)
var _ rest.Scoper = (*ProjectREST)(nil)
var _ rest.Getter = (*ProjectREST)(nil)
var _ rest.Lister = (*ProjectREST)(nil)
var _ rest.CreaterUpdater = (*ProjectREST)(nil)
var _ rest.GracefulDeleter = (*ProjectREST)(nil)
var _ rest.CollectionDeleter = (*ProjectREST)(nil)
var _ rest.Watcher = (*ProjectREST)(nil)
var _ rest.StandardStorage = (*ProjectREST)(nil)

func (p *ProjectREST) New() runtime.Object {
	return &Project{}
}

func (p *ProjectREST) NamespaceScoped() bool {
	return true
}

func (p *ProjectREST) NewList() runtime.Object {
	return &ProjectList{}
}

func (p *ProjectREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	uProject, err := p.dynamicRI.Namespace(request.NamespaceValue(ctx)).Get(ctx, name, *options)
	if err != nil {
		return nil, err
	}

	project, err := ConvertFromUnstructuredStorageV1Alpha1Project(uProject, p.scheme)
	if err != nil {
		return nil, err
	}
	if err := checkMembership(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}

func (p *ProjectREST) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	opts := &metav1.ListOptions{}
	if err := p.scheme.Convert(options, opts, nil); err != nil {
		return nil, err
	}
	projects, err := p.dynamicRI.Namespace(request.NamespaceValue(ctx)).List(ctx, *opts)
	if err != nil {
		return nil, err
	}
	spl, err := ConvertFromUnstructuredStorageV1Alpha1ProjectList(projects, p.scheme)
	if err != nil {
		return nil, err
	}

	lst := spl.Items
	spl.Items = nil
	for _, it := range lst {
		visible, err := isMember(ctx, &it)
		if err != nil {
			return nil, err
		}
		if visible {
			spl.Items = append(spl.Items, it)
		}
	}
	return spl, nil
}

func (o *ProjectREST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return rest.NewDefaultTableConvertor(Resource(externalProjectResource)).ConvertToTable(ctx, object, tableOptions)
}

func (p *ProjectREST) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	a, err := filters.GetAuthorizerAttributes(ctx)
	if err != nil {
		return nil, err
	}
	project := obj.(*Project)
	if err := createValidation(ctx, obj); err != nil {
		return nil, err
	}

	// Here we're not using checkOwnership since we're returning different error.
	// User should always include himself/herself in the Owners list, otherwise, we return BadRequest error to
	// indicate the request is invalid and cannot be processed.
	isOwner, err := containsUser(ctx, project.Spec.Owners)
	if err != nil {
		return nil, err
	}

	if !isOwner {
		return nil, apierrors.NewBadRequest("cannot create project you're not the owner of")
	}
	u, err := ConvertToUnstructuredStorageV1Alpha1Project(project, p.scheme)
	if err != nil {
		return nil, err
	}

	var subresource []string
	if a.GetSubresource() != "" {
		subresource = append(subresource, a.GetSubresource())
	}

	ret, err := p.dynamicRI.Namespace(request.NamespaceValue(ctx)).Create(ctx, u, *options, subresource...)
	if err != nil {
		return nil, err
	}

	obj, err = ConvertFromUnstructuredStorageV1Alpha1Project(ret, p.scheme)
	return obj, err
}

func (p *ProjectREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	a, err := filters.GetAuthorizerAttributes(ctx)
	if err != nil {
		return nil, false, err
	}
	preconditions := objInfo.Preconditions()
	rv := ""
	if preconditions != nil && preconditions.ResourceVersion != nil {
		rv = *preconditions.ResourceVersion
	}
	objUntyped, err := p.Get(ctx, name, &metav1.GetOptions{
		TypeMeta:        options.TypeMeta,
		ResourceVersion: rv,
	})
	if err != nil {
		return nil, false, err
	}
	oldObj := objUntyped.(*Project)
	if preconditions != nil && preconditions.UID != nil && oldObj.UID != *preconditions.UID {
		return nil, false, fmt.Errorf("UID differs, precondition UID: %s, found %s", *preconditions.UID, oldObj.UID)
	}
	if err := createValidation(ctx, oldObj); err != nil {
		return nil, false, err
	}
	if err := checkOwnership(ctx, oldObj); err != nil {
		return nil, false, err
	}
	newObj, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil {
		return nil, false, err
	}
	if err := updateValidation(ctx, newObj, oldObj); err != nil {
		return nil, false, err
	}

	u, err := ConvertToUnstructuredStorageV1Alpha1Project(newObj.(*Project), p.scheme)
	if err != nil {
		return nil, false, err
	}

	var subresource []string
	if a.GetSubresource() != "" {
		subresource = append(subresource, a.GetSubresource())
	}
	u, err = p.dynamicRI.Namespace(request.NamespaceValue(ctx)).Update(ctx, u, *options, subresource...)
	if err != nil {
		return nil, false, err
	}

	retObj, err := ConvertFromUnstructuredStorageV1Alpha1Project(u, p.scheme)
	if err != nil {
		return nil, false, err
	}
	return retObj, false, nil
}

func (p *ProjectREST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	obj, err := p.Get(ctx, name, &metav1.GetOptions{})
	if err != nil {
		return nil, false, err
	}
	if err := deleteValidation(ctx, obj); err != nil {
		return obj, false, err
	}
	if err := checkOwnership(ctx, obj.(*Project)); err != nil {
		return nil, false, err
	}
	err = p.dynamicRI.Namespace(request.NamespaceValue(ctx)).Delete(ctx, name, *options)
	return obj, false, err
}

func (p *ProjectREST) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions, listOptions *internalversion.ListOptions) (runtime.Object, error) {
	projects, err := p.List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	for _, project := range projects.(*ProjectList).Items {
		if err := checkOwnership(ctx, &project); err != nil {
			return nil, err
		}
	}
	opts := &metav1.ListOptions{}
	if err := p.scheme.Convert(listOptions, opts, nil); err != nil {
		return nil, err
	}
	if err := p.dynamicRI.Namespace(request.NamespaceValue(ctx)).DeleteCollection(ctx, *options, *opts); err != nil {
		return nil, err
	}
	return projects, nil
}

func (p *ProjectREST) Watch(ctx context.Context, options *internalversion.ListOptions) (watch.Interface, error) {
	opts := &metav1.ListOptions{}
	if err := p.scheme.Convert(options, opts, nil); err != nil {
		return nil, err
	}
	wi, err := p.dynamicRI.Namespace(request.NamespaceValue(ctx)).Watch(ctx, *opts)
	if err != nil {
		return nil, err
	}
	res := make(chan watch.Event)
	pw := watch.NewProxyWatcher(res)
	go func() {
		defer wi.Stop()
		defer close(res)
		for {
			select {
			case <-pw.StopChan():
				return
			case ev, ok := <-wi.ResultChan():
				if !ok {
					// channel closed
					return
				}
				if ev.Type == watch.Error {
					res <- ev
					return
				}
				project, err := ConvertFromUnstructuredStorageV1Alpha1Project(ev.Object.(*unstructured.Unstructured), p.scheme)
				if err != nil {
					res <- internalErrorWatchEvent(err)
					return
				}
				ev.Object = project
				visible, err := isMember(ctx, project)
				if err != nil {
					res <- internalErrorWatchEvent(err)
					return
				}
				if visible {
					res <- ev
				}
			}
		}
	}()
	return pw, nil
}
