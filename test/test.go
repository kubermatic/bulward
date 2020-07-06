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

package test

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/kubermatic/utils/pkg/testutil"
)

var (
	testScheme = scheme.Scheme
)

func TryUpdateUntil(ctx context.Context, cl *testutil.RecordingClient, obj runtime.Object, updateFn func() error) error {
	updateCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return wait.PollUntil(time.Second, func() (done bool, err error) {
		if err := testutil.WaitUntilFound(updateCtx, cl, obj); err != nil {
			return false, err
		}
		if err := updateFn(); err != nil {
			return false, err
		}
		if err := cl.Update(updateCtx, obj); err != nil {
			if errors.IsConflict(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}, updateCtx.Done())
}
