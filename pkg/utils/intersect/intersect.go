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
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func PolicyRules(rules1, rules2 []rbacv1.PolicyRule) []rbacv1.PolicyRule {
	var newRules []rbacv1.PolicyRule
	for _, r1 := range rules1 {
		for _, r2 := range rules2 {
			rule := PolicyRule(&r1, &r2)
			if rule != nil {
				newRules = append(newRules, *rule)
			}
		}
	}
	return newRules
}

func PolicyRule(rule1, rule2 *rbacv1.PolicyRule) *rbacv1.PolicyRule {
	verbs := intersectVerbs(rule1.Verbs, rule2.Verbs)
	if len(verbs) == 0 {
		return nil
	}
	policyRule := &rbacv1.PolicyRule{
		Verbs: verbs,
	}
	apiGroups := intersectAPIGroups(rule1.APIGroups, rule2.APIGroups)
	resources := intersectResources(rule1.Resources, rule2.Resources)
	resourceNames := intersectResourceNames(rule1.ResourceNames, rule2.ResourceNames)
	nonResourceURLs := intersectNonResourceURLs(rule1.NonResourceURLs, rule2.NonResourceURLs)
	var isValid bool
	if len(apiGroups) != 0 && len(resources) != 0 {
		// This rule is a valid resource rule.
		policyRule.APIGroups = apiGroups
		policyRule.Resources = resources
		policyRule.ResourceNames = resourceNames
		isValid = true
	}
	if len(nonResourceURLs) != 0 {
		// This rule is a valid non-resource rule.
		policyRule.NonResourceURLs = nonResourceURLs
		isValid = true
	}
	if isValid {
		return policyRule
	}
	return nil
}

func intersectVerbs(verbs1, verbs2 []string) []string {
	for _, verb := range verbs1 {
		if verb == rbacv1.VerbAll {
			return verbs2
		}
	}
	for _, verb := range verbs2 {
		if verb == rbacv1.VerbAll {
			return verbs1
		}
	}
	return intersection(verbs1, verbs2)
}

func intersectAPIGroups(apiGroups1, apiGroups2 []string) []string {
	for _, apiGroup := range apiGroups1 {
		if apiGroup == rbacv1.APIGroupAll {
			return apiGroups2
		}
	}
	for _, apiGroup := range apiGroups2 {
		if apiGroup == rbacv1.APIGroupAll {
			return apiGroups1
		}
	}
	return intersection(apiGroups1, apiGroups2)
}

func intersectResources(resources1, resources2 []string) []string {
	for _, resource := range resources1 {
		if resource == rbacv1.ResourceAll {
			return resources2
		}
	}
	for _, resource := range resources2 {
		if resource == rbacv1.ResourceAll {
			return resources1
		}
	}
	return intersection(resources1, resources2)
}

func intersectResourceNames(resourceNames1, resourceNames2 []string) []string {
	if len(resourceNames1) == 0 {
		return resourceNames2
	}
	if len(resourceNames2) == 0 {
		return resourceNames1
	}
	return intersection(resourceNames1, resourceNames2)
}

func intersectNonResourceURLs(nonResourceURLs1, nonResourceURLs2 []string) []string {
	for _, nonResourceURL := range nonResourceURLs1 {
		if nonResourceURL == rbacv1.NonResourceAll {
			return nonResourceURLs2
		}
	}
	for _, nonResourceURL := range nonResourceURLs2 {
		if nonResourceURL == rbacv1.NonResourceAll {
			return nonResourceURLs1
		}
	}
	return intersection(nonResourceURLs1, nonResourceURLs2)
}

func intersection(s1, s2 []string) (inter []string) {
	return sets.NewString(s1...).Intersection(sets.NewString(s2...)).List()
}
