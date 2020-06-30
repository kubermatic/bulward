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
	"fmt"
	"os"

	"github.com/spf13/cobra"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/client-go/dynamic"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/apiserver-builder-alpha/pkg/apiserver"
	"sigs.k8s.io/apiserver-builder-alpha/pkg/builders"
	"sigs.k8s.io/apiserver-builder-alpha/pkg/cmd/server"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/kubermatic/bulward/pkg/apis"
	apiserverapi "github.com/kubermatic/bulward/pkg/apis/apiserver"
	apiserverv1alpha1 "github.com/kubermatic/bulward/pkg/apis/apiserver/v1alpha1"
	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
	"github.com/kubermatic/bulward/pkg/openapi"
)

type flags struct {
	bulwardSystemNamespace string
	metricsAddr            string
}

const (
	componentAPIServer = "apiserver"
)

func init() {
	// due to apiserver-builder-alpha usage we must use the following scheme
	utilruntime.Must(clientgoscheme.AddToScheme(builders.Scheme))
	utilruntime.Must(corev1alpha1.AddToScheme(builders.Scheme))
	utilruntime.Must(apiserverapi.AddToScheme(builders.Scheme))
	utilruntime.Must(apiserverv1alpha1.AddToScheme(builders.Scheme))
	utilruntime.Must(apiserverapi.Corev1alpha1RegisterConversion(builders.Scheme))
}

// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations;validatingwebhookconfigurations,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create
// +kubebuilder:rbac:groups=bulward.io,resources=organizations,verbs=create;get;list;watch;update;patch;delete

func NewAPIServerCommand() *cobra.Command {
	log := ctrl.Log.WithName("apiserver")
	_ = log
	signalCh := genericapiserver.SetupSignalHandler()
	flags := &flags{}
	cmd, _ := server.NewCommandStartServer(
		"",
		os.Stdout,
		os.Stderr,
		apis.GetAllApiBuilders(),
		signalCh,
		"apiserver",
		"v0",
		func(apiServer *apiserver.Config) error {
			apiServer.RecommendedConfig.HealthzChecks = filterHealthChecks(apiServer.RecommendedConfig.HealthzChecks, "etcd")
			apiServer.RecommendedConfig.LivezChecks = filterHealthChecks(apiServer.RecommendedConfig.LivezChecks, "etcd")
			apiServer.RecommendedConfig.ReadyzChecks = filterHealthChecks(apiServer.RecommendedConfig.ReadyzChecks, "etcd")
			apiServer.RecommendedConfig.RESTOptionsGetter = nil // we're not using etcd nor anything like this
			return nil
		},
	)
	cmd.Use = componentAPIServer
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// from server.StartApiServerWithOptions()
		// which isn't called during server.NewCommandStartServer
		server.GetOpenApiDefinition = openapi.GetOpenAPIDefinitions

		if flags.bulwardSystemNamespace == "" {
			return fmt.Errorf("--bulward-system-namespace or ENVVAR BULWARD_NAMESPACE must be set")
		}
		loader := clientcmd.NewDefaultClientConfigLoadingRules()
		loader.ExplicitPath = cmd.Flag("kubeconfig").Value.String()
		cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loader,
			&clientcmd.ConfigOverrides{},
		).ClientConfig()
		if err != nil {
			return err
		}
		mapper, err := apiutil.NewDynamicRESTMapper(cfg)
		if err != nil {
			return err
		}
		k8sClient, err := client.New(cfg, client.Options{
			Scheme: builders.Scheme,
			Mapper: mapper,
		})
		if err != nil {
			return err
		}
		dynamicClient, err := dynamic.NewForConfig(cfg)
		if err != nil {
			return err
		}
		if err := apiserverapi.OrganizationRESTSingleton.InjectMapper(mapper); err != nil {
			return err
		}
		if err := apiserverapi.OrganizationRESTSingleton.InjectClient(k8sClient); err != nil {
			return err
		}
		if err := apiserverapi.OrganizationRESTSingleton.InjectDynamicClient(dynamicClient); err != nil {
			return err
		}
		if err := apiserverapi.OrganizationRESTSingleton.InjectScheme(builders.Scheme); err != nil {
			return err
		}
		return nil
	}
	cmd.Flags().StringVar(&flags.metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	cmd.Flags().StringVar(&flags.bulwardSystemNamespace, "bulward-system-namespace", os.Getenv("BULWARD_NAMESPACE"), "The namespace that Bulward controller manager deploys to.")
	return cmd
}
func filterHealthChecks(in []healthz.HealthChecker, exclude string) []healthz.HealthChecker {
	var out []healthz.HealthChecker
	for _, it := range in {
		if it.Name() != exclude {
			out = append(out, it)
		}
	}
	return out
}
