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

package intersect

import (
	"testing"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestIntersectPolicyRule(t *testing.T) {
	tests := []struct {
		description                            string
		rule1, rule2, expectedIntersectionRule *rbacv1.PolicyRule
	}{
		{
			description: "apiGroups don't match",
			rule1: &rbacv1.PolicyRule{
				APIGroups: []string{"apiserver.bulward.io"},
				Resources: []string{"projects"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			rule2: &rbacv1.PolicyRule{
				APIGroups: []string{"storage.bulward.io"},
				Resources: []string{"projects", "organizations"},
				Verbs:     []string{"get", "list", "watch", "create", "update"},
			},
			expectedIntersectionRule: nil,
		},
		{
			description: "resources don't match",
			rule1: &rbacv1.PolicyRule{
				APIGroups: []string{"apiserver.bulward.io"},
				Resources: []string{"projects"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			rule2: &rbacv1.PolicyRule{
				APIGroups: []string{"apiserver.bulward.io"},
				Resources: []string{"organizations"},
				Verbs:     []string{"get", "list", "watch", "create", "update"},
			},
			expectedIntersectionRule: nil,
		},
		{
			description: "verbs don't match",
			rule1: &rbacv1.PolicyRule{
				APIGroups: []string{"apiserver.bulward.io"},
				Resources: []string{"projects"},
				Verbs:     []string{"patch", "delete"},
			},
			rule2: &rbacv1.PolicyRule{
				APIGroups: []string{"apiserver.bulward.io"},
				Resources: []string{"organizations"},
				Verbs:     []string{"get", "list", "watch", "create", "update"},
			},
			expectedIntersectionRule: nil,
		},
		{
			description: "apiGroup subset",
			rule1: &rbacv1.PolicyRule{
				APIGroups: []string{"apiserver.bulward.io", "storage.bulward.io"},
				Resources: []string{rbacv1.ResourceAll},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			rule2: &rbacv1.PolicyRule{
				APIGroups: []string{"apiserver.bulward.io"},
				Resources: []string{"projects"},
				Verbs:     []string{"get", "list", "watch", "create", "update"},
			},
			expectedIntersectionRule: &rbacv1.PolicyRule{
				APIGroups: []string{"apiserver.bulward.io"},
				Resources: []string{"projects"},
				Verbs:     []string{"create", "get", "list", "update", "watch"},
			},
		},
		{
			description: "apiGroup subset",
			rule1: &rbacv1.PolicyRule{
				APIGroups:     []string{"apiserver.bulward.io", "storage.bulward.io"},
				Resources:     []string{rbacv1.ResourceAll},
				ResourceNames: []string{},
				Verbs:         []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			rule2: &rbacv1.PolicyRule{
				APIGroups:     []string{"apiserver.bulward.io"},
				Resources:     []string{"projects"},
				ResourceNames: []string{"aaa", "bbb"},
				Verbs:         []string{"get", "list", "watch", "create", "update"},
			},
			expectedIntersectionRule: &rbacv1.PolicyRule{
				APIGroups:     []string{"apiserver.bulward.io"},
				Resources:     []string{"projects"},
				ResourceNames: []string{"aaa", "bbb"},
				Verbs:         []string{"create", "get", "list", "update", "watch"},
			},
		},
		{
			description: "non-resource rule",
			rule1: &rbacv1.PolicyRule{
				Verbs:           []string{"get", "list", "watch", "create", "update", "patch", "delete"},
				NonResourceURLs: []string{rbacv1.NonResourceAll},
			},
			rule2: &rbacv1.PolicyRule{
				Verbs:           []string{"get", "list", "watch", "create", "update"},
				NonResourceURLs: []string{"resource/aaa"},
			},
			expectedIntersectionRule: &rbacv1.PolicyRule{
				Verbs:           []string{"create", "get", "list", "update", "watch"},
				NonResourceURLs: []string{"resource/aaa"},
			},
		},
	}
	for _, test := range tests {
		assert.Equal(t, test.expectedIntersectionRule, PolicyRule(test.rule1, test.rule2))
	}
}

func TestIntersectVerbs(t *testing.T) {
	tests := []struct {
		verbs1, verbs2, expectedIntersection []string
	}{
		{
			verbs1:               []string{"aaa"},
			verbs2:               []string{rbacv1.VerbAll},
			expectedIntersection: []string{"aaa"},
		},
		{
			verbs1:               []string{},
			verbs2:               []string{rbacv1.VerbAll},
			expectedIntersection: []string{},
		},
		{
			verbs1:               []string{"ccc"},
			verbs2:               []string{"bbb", "aaa"},
			expectedIntersection: []string{},
		},
		{
			verbs1:               []string{"ccc", "aaa"},
			verbs2:               []string{"bbb", "aaa"},
			expectedIntersection: []string{"aaa"},
		},
	}

	for _, test := range tests {
		assert.ElementsMatch(t, test.expectedIntersection, intersectVerbs(test.verbs1, test.verbs2))
	}
}

func TestIntersectAPIGroups(t *testing.T) {
	tests := []struct {
		apiGroups1, apiGroups2, expectedIntersection []string
	}{
		{
			apiGroups1:           []string{"aaa.io"},
			apiGroups2:           []string{rbacv1.APIGroupAll},
			expectedIntersection: []string{"aaa.io"},
		},
		{
			apiGroups1:           []string{},
			apiGroups2:           []string{rbacv1.APIGroupAll},
			expectedIntersection: []string{},
		},
		{
			apiGroups1:           []string{"ccc.io"},
			apiGroups2:           []string{"bbb.io", "aaa.io"},
			expectedIntersection: []string{},
		},
		{
			apiGroups1:           []string{"ccc.io", "aaa.io"},
			apiGroups2:           []string{"bbb.io", "aaa.io"},
			expectedIntersection: []string{"aaa.io"},
		},
	}

	for _, test := range tests {
		assert.ElementsMatch(t, test.expectedIntersection, intersectAPIGroups(test.apiGroups1, test.apiGroups2))
	}
}

func TestIntersectResources(t *testing.T) {
	tests := []struct {
		resources1, resources2, expectedIntersection []string
	}{
		{
			resources1:           []string{"aaa"},
			resources2:           []string{rbacv1.ResourceAll},
			expectedIntersection: []string{"aaa"},
		},
		{
			resources1:           []string{},
			resources2:           []string{rbacv1.ResourceAll},
			expectedIntersection: []string{},
		},
		{
			resources1:           []string{"ccc"},
			resources2:           []string{"bbb", "aaa"},
			expectedIntersection: []string{},
		},
		{
			resources1:           []string{"ccc", "aaa"},
			resources2:           []string{"bbb", "aaa"},
			expectedIntersection: []string{"aaa"},
		},
	}

	for _, test := range tests {
		assert.ElementsMatch(t, test.expectedIntersection, intersectResources(test.resources1, test.resources2))
	}
}

func TestIntersectResourceNames(t *testing.T) {
	tests := []struct {
		resourceNames1, resourceNames2, expectedIntersection []string
	}{
		{
			resourceNames1:       []string{"aaa"},
			resourceNames2:       []string{},
			expectedIntersection: []string{"aaa"},
		},
		{
			resourceNames1:       []string{},
			resourceNames2:       []string{"bbb"},
			expectedIntersection: []string{"bbb"},
		},
		{
			resourceNames1:       []string{"ccc"},
			resourceNames2:       []string{"bbb", "aaa"},
			expectedIntersection: []string{},
		},
		{
			resourceNames1:       []string{"ccc", "aaa"},
			resourceNames2:       []string{"bbb", "aaa"},
			expectedIntersection: []string{"aaa"},
		},
	}

	for _, test := range tests {
		assert.ElementsMatch(t, test.expectedIntersection, intersectResourceNames(test.resourceNames1, test.resourceNames2))
	}
}

func TestIntersectNonResourceURLs(t *testing.T) {
	tests := []struct {
		nonResourceURLs1, nonResourceURLs2, expectedIntersection []string
	}{
		{
			nonResourceURLs1:     []string{"aaa"},
			nonResourceURLs2:     []string{rbacv1.NonResourceAll},
			expectedIntersection: []string{"aaa"},
		},
		{
			nonResourceURLs1:     []string{},
			nonResourceURLs2:     []string{rbacv1.NonResourceAll},
			expectedIntersection: []string{},
		},
		{
			nonResourceURLs1:     []string{"ccc"},
			nonResourceURLs2:     []string{"bbb", "aaa"},
			expectedIntersection: []string{},
		},
		{
			nonResourceURLs1:     []string{"ccc", "aaa"},
			nonResourceURLs2:     []string{"bbb", "aaa"},
			expectedIntersection: []string{"aaa"},
		},
	}

	for _, test := range tests {
		assert.ElementsMatch(t, test.expectedIntersection, intersectNonResourceURLs(test.nonResourceURLs1, test.nonResourceURLs2))
	}
}

func TestIntersection(t *testing.T) {
	tests := []struct {
		s1, s2, expectedIntersection []string
	}{
		{
			s1:                   []string{"aaa"},
			s2:                   []string{},
			expectedIntersection: []string{},
		},
		{
			s1:                   []string{},
			s2:                   []string{"bbb"},
			expectedIntersection: []string{},
		},
		{
			s1:                   []string{"aaa"},
			s2:                   []string{"bbb"},
			expectedIntersection: []string{},
		},
		{
			s1:                   []string{"aaa", "bbb"},
			s2:                   []string{"aaa"},
			expectedIntersection: []string{"aaa"},
		},
		{
			s1:                   []string{"aaa", "bbb", "ccc"},
			s2:                   []string{"aaa", "ddd"},
			expectedIntersection: []string{"aaa"},
		},
		{
			s1:                   []string{"aaa", "bbb", "ccc", "ddd"},
			s2:                   []string{"aaa", "ddd"},
			expectedIntersection: []string{"aaa", "ddd"},
		},
	}

	for _, test := range tests {
		assert.ElementsMatch(t, test.expectedIntersection, intersection(test.s1, test.s2))
	}
}
