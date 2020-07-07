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
	"net/http"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/filters"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	storagev1alpha1 "github.com/kubermatic/bulward/pkg/apis/storage/v1alpha1"
)

const (
	internalOrganizationResource = "organizations"
	externalOrganizationResource = "organizations"
)

var (
	qualifiedResource = schema.GroupResource{
		Group:    SchemeGroupVersion.Group,
		Resource: externalOrganizationResource,
	}
)

// +k8s:deepcopy-gen=false
type OrganizationREST struct {
	client    client.Client
	dynamicRI dynamic.ResourceInterface
	mapper    meta.RESTMapper
	scheme    *runtime.Scheme
}

var OrganizationRESTSingleton = &OrganizationREST{}

func NewOrganizationREST(_ generic.RESTOptionsGetter) rest.Storage {
	return OrganizationRESTSingleton
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
	if o.dynamicRI != nil {
		return fmt.Errorf("dynamicRI already injected")
	}
	o.dynamicRI = dynamic.Resource(storagev1alpha1.GroupVersion.WithResource(internalOrganizationResource))
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
var _ rest.CreaterUpdater = (*OrganizationREST)(nil)
var _ rest.GracefulDeleter = (*OrganizationREST)(nil)
var _ rest.CollectionDeleter = (*OrganizationREST)(nil)
var _ rest.Watcher = (*OrganizationREST)(nil)
var _ rest.StandardStorage = (*OrganizationREST)(nil)

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
	uOrg, err := o.dynamicRI.Get(ctx, name, *options)
	if err != nil {
		return nil, err
	}

	org, err := ConvertFromUnstructuredCoreV1Alpha1(uOrg, o.scheme)
	if err != nil {
		return nil, err
	}
	visible, err := o.isVisible(ctx, org)
	if err != nil {
		return nil, err
	}
	if !visible {
		return nil, apierrors.NewNotFound(qualifiedResource, name)
	}
	return org, nil
}

func (o *OrganizationREST) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	opts := &metav1.ListOptions{}
	if err := o.scheme.Convert(options, opts, nil); err != nil {
		return nil, err
	}
	orgs, err := o.dynamicRI.List(ctx, *opts)
	if err != nil {
		return nil, err
	}
	sol, err := ConvertFromUnstructuredCoreV1Alpha1List(orgs, o.scheme)
	if err != nil {
		return nil, err
	}

	lst := sol.Items
	sol.Items = nil
	for _, it := range lst {
		visible, err := o.isVisible(ctx, &it)
		if err != nil {
			return nil, err
		}
		if visible {
			sol.Items = append(sol.Items, it)
		}
	}
	return sol, nil
}

func (o *OrganizationREST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return rest.NewDefaultTableConvertor(Resource(externalOrganizationResource)).ConvertToTable(ctx, object, tableOptions)
}

func (o *OrganizationREST) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	a, err := filters.GetAuthorizerAttributes(ctx)
	if err != nil {
		return nil, err
	}
	org := obj.(*Organization)
	if err := createValidation(ctx, obj); err != nil {
		return nil, err
	}
	visible, err := o.isVisible(ctx, org)
	if err != nil {
		return nil, err
	}
	if !visible {
		return nil, apierrors.NewBadRequest("cannot create organization you're not the owner of")
	}
	u, err := ConvertToUnstructuredCoreV1Alpha1Organization(org, o.scheme)
	if err != nil {
		return nil, err
	}

	var subresource []string
	if a.GetSubresource() != "" {
		subresource = append(subresource, a.GetSubresource())
	}
	ret, err := o.dynamicRI.Create(ctx, u, *options, subresource...)
	if err != nil {
		return nil, err
	}
	obj, err = ConvertFromUnstructuredCoreV1Alpha1(ret, o.scheme)
	return obj, err
}

func (o *OrganizationREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	a, err := filters.GetAuthorizerAttributes(ctx)
	if err != nil {
		return nil, false, err
	}
	preconditions := objInfo.Preconditions()
	rv := ""
	if preconditions != nil && preconditions.ResourceVersion != nil {
		rv = *preconditions.ResourceVersion
	}
	objUntyped, err := o.Get(ctx, name, &metav1.GetOptions{
		TypeMeta:        options.TypeMeta,
		ResourceVersion: rv,
	})
	if err != nil {
		return nil, false, err
	}
	oldObj := objUntyped.(*Organization)
	if preconditions != nil && preconditions.UID != nil && oldObj.UID != *preconditions.UID {
		return nil, false, fmt.Errorf("UID differs, precondition UID: %s, found %s", *preconditions.UID, oldObj.UID)
	}
	if err := createValidation(ctx, oldObj); err != nil {
		return nil, false, err
	}
	visible, err := o.isVisible(ctx, oldObj)
	if err != nil {
		return nil, false, err
	}
	if !visible {
		return nil, false, apierrors.NewNotFound(qualifiedResource, name)
	}

	newObj, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil {
		return nil, false, err
	}
	if err := updateValidation(ctx, newObj, oldObj); err != nil {
		return nil, false, err
	}

	u, err := ConvertToUnstructuredCoreV1Alpha1Organization(newObj.(*Organization), o.scheme)
	if err != nil {
		return nil, false, err
	}

	var subresource []string
	if a.GetSubresource() != "" {
		subresource = append(subresource, a.GetSubresource())
	}
	u, err = o.dynamicRI.Update(ctx, u, *options, subresource...)
	if err != nil {
		return nil, false, err
	}

	retObj, err := ConvertFromUnstructuredCoreV1Alpha1(u, o.scheme)
	if err != nil {
		return nil, false, err
	}
	return retObj, false, nil
}

func (o *OrganizationREST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	obj, err := o.Get(ctx, name, &metav1.GetOptions{})
	if err != nil {
		return nil, false, err
	}
	if err := deleteValidation(ctx, obj); err != nil {
		return obj, false, err
	}
	err = o.dynamicRI.Delete(ctx, name, *options)
	return obj, false, err
}

func (o *OrganizationREST) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions, listOptions *internalversion.ListOptions) (runtime.Object, error) {
	opts := &metav1.ListOptions{}
	if err := o.scheme.Convert(listOptions, opts, nil); err != nil {
		return nil, err
	}
	err := o.dynamicRI.DeleteCollection(ctx, *options, *opts)
	return nil, err
}

func (o *OrganizationREST) Watch(ctx context.Context, options *internalversion.ListOptions) (watch.Interface, error) {
	opts := &metav1.ListOptions{}
	if err := o.scheme.Convert(options, opts, nil); err != nil {
		return nil, err
	}
	wi, err := o.dynamicRI.Watch(ctx, *opts)
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
				org, err := ConvertFromUnstructuredCoreV1Alpha1(ev.Object.(*unstructured.Unstructured), o.scheme)
				if err != nil {
					res <- internalErrorWatchEvent(err)
					return
				}
				ev.Object = org
				visible, err := o.isVisible(ctx, org)
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

func internalErrorWatchEvent(err error) watch.Event {
	return watch.Event{
		Type: watch.Error,
		Object: &metav1.Status{
			Status:  "error",
			Message: err.Error(),
			Reason:  metav1.StatusReasonInternalError,
			Code:    http.StatusInternalServerError,
		},
	}
}

func (o *OrganizationREST) isVisible(ctx context.Context, organization *Organization) (bool, error) {
	attrs, err := filters.GetAuthorizerAttributes(ctx)
	if err != nil {
		return false, err
	}
	user := attrs.GetUser()
	if user == nil {
		klog.Warning("unknown user, you may running API extension server with --delegated-auth=false")
		return true, nil
	}

	for _, sub := range append(
		organization.Status.Members,
		// This is important for seeing organizations you own before controller syncs status
		// otherwise a watch misses create event
		organization.Spec.Owners...,
	) {
		switch sub.Kind {
		case rbacv1.UserKind:
			if sub.Name == user.GetName() {
				return true, nil
			}
		case rbacv1.GroupKind:
			for _, grp := range user.GetGroups() {
				if sub.Name == grp {
					return true, nil
				}
			}
		case rbacv1.ServiceAccountKind:
			if fmt.Sprintf("system:serviceaccount:%s:%s", sub.Namespace, sub.Name) == user.GetName() {
				return true, nil
			}
		default:
			return false, fmt.Errorf("unknown subject's kind: %s, %v in organization %s", sub.Kind, sub, organization.Name)
		}
	}
	return false, err
}
