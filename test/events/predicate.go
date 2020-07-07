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

package events

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/watch"
)

type Predicate func(event watch.Event) (bool, error)

func IsObjectName(name string) Predicate {
	return func(event watch.Event) (bool, error) {
		obj, err := meta.Accessor(event.Object)
		if err != nil {
			return false, err
		}
		return obj.GetName() == name, nil
	}
}

func IsType(eventType watch.EventType) Predicate {
	return func(event watch.Event) (bool, error) {
		return event.Type == eventType, nil
	}
}

func AllOf(predicates ...Predicate) Predicate {
	return func(event watch.Event) (bool, error) {
		for _, p := range predicates {
			ok, err := p(event)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
		return true, nil
	}
}

func AnyOf(predicates ...Predicate) Predicate {
	return func(event watch.Event) (bool, error) {
		for _, p := range predicates {
			ok, err := p(event)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}
}
