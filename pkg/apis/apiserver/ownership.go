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

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/filters"
	"k8s.io/klog"
)

// +k8s:deepcopy-gen=false
type OwnableResourceWithMembership interface {
	GetName() string
	GetOwners() []rbacv1.Subject
	GetMembers() []rbacv1.Subject
	GetQualifiedResource() schema.GroupResource
}

func (p *Project) GetOwners() []rbacv1.Subject {
	return p.Spec.Owners
}

func (p *Project) GetMembers() []rbacv1.Subject {
	return p.Status.Members
}

func (p *Project) GetQualifiedResource() schema.GroupResource {
	return schema.GroupResource{
		Group:    SchemeGroupVersion.Group,
		Resource: externalProjectResource,
	}
}

func (p *Organization) GetOwners() []rbacv1.Subject {
	return p.Spec.Owners
}

func (p *Organization) GetMembers() []rbacv1.Subject {
	return p.Status.Members
}

func (p *Organization) GetQualifiedResource() schema.GroupResource {
	return schema.GroupResource{
		Group:    SchemeGroupVersion.Group,
		Resource: externalOrganizationResource,
	}
}

// checkOwnership checks if the calling user is owner of the resource, and if not returns appropriate error:
// NotFound if non-member, Forbidden if member
func checkOwnership(ctx context.Context, ownRes OwnableResourceWithMembership) error {
	attrs, err := filters.GetAuthorizerAttributes(ctx)
	if err != nil {
		return err
	}
	if err := checkMembership(ctx, ownRes); err != nil {
		return err
	}
	isOwner, err := containsUser(ctx, ownRes.GetOwners())
	if err != nil {
		return err
	}
	if !isOwner {
		return apierrors.NewForbidden(
			ownRes.GetQualifiedResource(),
			ownRes.GetName(),
			fmt.Errorf("ownership is required for %s operation", attrs.GetVerb()),
		)
	}
	return nil
}

// checkMembership checks if the calling user is project member, and if not returns NotFound error
func checkMembership(ctx context.Context, ownRes OwnableResourceWithMembership) error {
	visible, err := isMember(ctx, ownRes)
	if err != nil {
		return err
	}
	if !visible {
		return apierrors.NewNotFound(ownRes.GetQualifiedResource(), ownRes.GetName())
	}
	return nil
}

// isMember checks if the calling user is a resource member
func isMember(ctx context.Context, ownRes OwnableResourceWithMembership) (bool, error) {
	return containsUser(ctx,
		append(
			// This is important for seeing the resource you own before controller syncs status
			// otherwise a watch misses create event
			ownRes.GetOwners(),
			ownRes.GetMembers()...,
		),
	)
}

// containsUser checks whether the calling user is in the subject list
func containsUser(ctx context.Context, subjects []rbacv1.Subject) (bool, error) {
	attrs, err := filters.GetAuthorizerAttributes(ctx)
	if err != nil {
		return false, err
	}
	user := attrs.GetUser()
	if user == nil {
		klog.Warning("unknown user, you may running API extension server with --delegated-auth=false")
		return true, nil
	}

	for _, sub := range subjects {
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
			return false, fmt.Errorf("unknown subject's kind: %s, %v", sub.Kind, sub)
		}
	}
	return false, err
}
