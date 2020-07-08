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
	"flag"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	apiserverv1alpha1 "github.com/kubermatic/bulward/pkg/apis/apiserver/v1alpha1"
	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
	"github.com/kubermatic/utils/pkg/testutil"
)

var (
	cleanUpStrategy = string(testutil.CleanupOnSuccess)
	testScheme      = scheme.Scheme
)

func init() {
	utilruntime.Must(corev1alpha1.AddToScheme(testScheme))
	utilruntime.Must(apiserverv1alpha1.AddToScheme(testScheme))
	utilruntime.Must(scheme.AddToScheme(testScheme))
	flag.StringVar(&cleanUpStrategy, "clean-up-strategy", cleanUpStrategy, "clean up strategy")
}
