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

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"sigs.k8s.io/apiserver-builder-alpha/pkg/cmd/server"
	ctrl "sigs.k8s.io/controller-runtime"
)

type flags struct {
	bulwardSystemNamespace string
	metricsAddr            string
}

const (
	componentAPIServer = "apiserver"
)

func NewAPIServerCommand() *cobra.Command {
	log := ctrl.Log.WithName("manager")
	flags := &flags{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   componentAPIServer,
		Short: "deploy Bulward api server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(flags, log)
		},
	}
	cmd.Flags().StringVar(&flags.metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	cmd.Flags().StringVar(&flags.bulwardSystemNamespace, "bulward-system-namespace", os.Getenv("BULWARD_NAMESPACE"), "The namespace that Bulward controller manager deploys to.")
	return cmd
}

func run(flags *flags, log logr.Logger) error {
	if flags.bulwardSystemNamespace == "" {
		return fmt.Errorf("-bulward-system-namespace or ENVVAR BULWARD_NAMESPACE must be set")
	}

	log.Info("starting apiserver")
	version := "v0"
	err := server.StartApiServerWithOptions(&server.StartOptions{
		EtcdPath: "/registry/example.com",
		Title:    "Api",
		Version:  version,
	})
	if err != nil {
		return fmt.Errorf("starting apiserver: %w", err)
	}
	return nil
}
