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
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/healthz"
	"sigs.k8s.io/apiserver-builder-alpha/pkg/apiserver"
	"sigs.k8s.io/apiserver-builder-alpha/pkg/cmd/server"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kubermatic/bulward/pkg/apis"
	"github.com/kubermatic/bulward/pkg/openapi"
)

type flags struct {
	bulwardSystemNamespace string
	metricsAddr            string
}

const (
	componentAPIServer = "apiserver"
)

func NewAPIServerCommand() *cobra.Command {
	log := ctrl.Log.WithName("apiserver")
	_ = log
	signalCh := genericapiserver.SetupSignalHandler()
	flags := &flags{}
	cmd, opts := server.NewCommandStartServer(
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
	_ = opts
	cmd.Use = componentAPIServer
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		server.GetOpenApiDefinition = openapi.GetOpenAPIDefinitions
		if flags.bulwardSystemNamespace == "" {
			return fmt.Errorf("--bulward-system-namespace or ENVVAR BULWARD_NAMESPACE must be set")
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

