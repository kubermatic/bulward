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

	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/filters"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
)

const (
	internalOrganizationResouce = "internalorganizations"
)

// +k8s:deepcopy-gen=false
type OrganizationREST struct {
	client    client.Client
	dynamicRI dynamic.NamespaceableResourceInterface
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
	o.dynamicRI = dynamic.Resource(corev1alpha1.GroupVersion.WithResource(internalOrganizationResouce))
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
	visible, err := o.isVisible(ctx, name, "get")
	if err != nil {
		return nil, err
	}
	if !visible {
		return nil, fmt.Errorf("NotFound")
	}

	org, err := o.dynamicRI.Get(ctx, name, *options)
	if err != nil {
		return nil, err
	}
	return ConvertFromV1Alpha1Unstructured(org, o.scheme)
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
	sol, err := ConvertFromV1Alpha1UnstructuredList(orgs, o.scheme)
	if err != nil {
		return nil, err
	}

	lst := sol.Items
	sol.Items = nil
	for _, it := range lst {
		visible, err := o.isVisible(ctx, it.Name, "get")
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
	return rest.NewDefaultTableConvertor(Resource("organizations")).ConvertToTable(ctx, object, tableOptions)
}

func (o *OrganizationREST) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	a, err := filters.GetAuthorizerAttributes(ctx)
	if err != nil {
		return nil, err
	}
	if err := createValidation(ctx, obj); err != nil {
		return nil, err
	}
	u, err := ConvertToV1Alpha1Unstructured(obj.(*Organization), o.scheme)
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
	return ConvertFromV1Alpha1Unstructured(ret, o.scheme)
}

func (o *OrganizationREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	visible, err := o.isVisible(ctx, name, "delete")
	if err != nil {
		return nil, false, err
	}
	if !visible {
		return nil, false, fmt.Errorf("NotFound")
	}

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
	newObj, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil {
		return nil, false, err
	}
	if err := updateValidation(ctx, newObj, oldObj); err != nil {
		return nil, false, err
	}

	u, err := ConvertToV1Alpha1Unstructured(newObj.(*Organization), o.scheme)
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

	retObj, err := ConvertFromV1Alpha1Unstructured(u, o.scheme)
	if err != nil {
		return nil, false, err
	}
	return retObj, false, nil
}

func (o *OrganizationREST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	visible, err := o.isVisible(ctx, name, "delete")
	if err != nil {
		return nil, false, err
	}
	if !visible {
		return nil, false, fmt.Errorf("NotFound")
	}
	err = o.dynamicRI.Delete(ctx, name, *options)
	return nil, false, err
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
			case ev := <-wi.ResultChan():
				if ev.Type == watch.Error {
					res <- ev
					return
				}
				org, err := ConvertFromV1Alpha1Unstructured(ev.Object.(*unstructured.Unstructured), o.scheme)
				if err != nil {
					res <- internalErrorWatchEvent(err)
					return
				}
				ev.Object = org
				visible, err := o.isVisible(ctx, org.Name, "watch")
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

func (o *OrganizationREST) isVisible(ctx context.Context, organizationName string, verb string) (bool, error) {
	a, err := filters.GetAuthorizerAttributes(ctx)
	if err != nil {
		return false, err
	}
	user := a.GetUser()
	if user == nil {
		klog.Warning("user info missing; you're probably running API extension server with --delegated-auth=false")
		return true, nil
	}

	extra := make(map[string]authorizationv1.ExtraValue)
	for k, v := range user.GetExtra() {
		extra[k] = v
	}
	subjectAccessReview := &authorizationv1.SubjectAccessReview{Spec: authorizationv1.SubjectAccessReviewSpec{
		ResourceAttributes: &authorizationv1.ResourceAttributes{
			Namespace:   a.GetNamespace(),
			Verb:        verb,
			Group:       corev1alpha1.GroupVersion.Group,
			Version:     corev1alpha1.GroupVersion.Version,
			Resource:    internalOrganizationResouce,
			Subresource: a.GetSubresource(),
			Name:        organizationName,
		},
		NonResourceAttributes: nil,
		User:                  user.GetName(),
		Groups:                user.GetGroups(),
		Extra:                 extra,
		UID:                   user.GetUID(),
	},
	}
	if err := o.client.Create(ctx, subjectAccessReview); err != nil {
		return false, err
	}
	return subjectAccessReview.Status.Allowed, nil
}
