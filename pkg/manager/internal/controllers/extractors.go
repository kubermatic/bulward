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

package controllers

import (
	"sort"

	rbacv1 "k8s.io/api/rbac/v1"
)

func extractSubjects(rbs *rbacv1.RoleBindingList) []rbacv1.Subject {
	var subjects []rbacv1.Subject
	for _, rb := range rbs.Items {
		subjects = append(subjects, rb.Subjects...)
	}
	sort.Slice(subjects, func(i, j int) bool {
		a := subjects[i]
		b := subjects[j]
		return a.String() < b.String()
	})
	filteredSubjects := make([]rbacv1.Subject, 0, len(subjects))
	for i := range subjects {
		if i == 0 || subjects[i-1].String() != subjects[i].String() {
			filteredSubjects = append(filteredSubjects, subjects[i])
		}
	}
	return filteredSubjects
}
