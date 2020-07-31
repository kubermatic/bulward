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

package manager

import (
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"k8c.io/utils/pkg/util"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	corev1alpha1 "github.com/kubermatic/bulward/pkg/apis/core/v1alpha1"
	storagev1alpha1 "github.com/kubermatic/bulward/pkg/apis/storage/v1alpha1"
	"github.com/kubermatic/bulward/pkg/manager/internal/controllers"
)

type flags struct {
	bulwardSystemNamespace string
	metricsAddr            string
	healthAddr             string
	enableLeaderElection   bool
}

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corev1alpha1.AddToScheme(scheme))
	utilruntime.Must(storagev1alpha1.AddToScheme(scheme))
}

const (
	componentManager = "manager"
)

func NewManagerCommand() *cobra.Command {
	log := ctrl.Log.WithName("manager")
	flags := &flags{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   componentManager,
		Short: "deploy Bulward controller manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(flags, log)
		},
	}
	cmd.Flags().StringVar(&flags.metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	cmd.Flags().StringVar(&flags.healthAddr, "health-addr", ":9440", "The address the health endpoint binds to.")
	cmd.Flags().BoolVar(&flags.enableLeaderElection, "enable-leader-election", true,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	cmd.Flags().StringVar(&flags.bulwardSystemNamespace, "bulward-system-namespace", os.Getenv("BULWARD_NAMESPACE"), "The namespace that Bulward controller manager deploys to.")
	return util.CmdLogMixin(cmd)
}

func run(flags *flags, log logr.Logger) error {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      flags.metricsAddr,
		LeaderElection:          flags.enableLeaderElection,
		LeaderElectionID:        "main-controller-manager",
		LeaderElectionNamespace: flags.bulwardSystemNamespace,
		Port:                    9443,
		HealthProbeBindAddress:  flags.healthAddr,
	})
	if err != nil {
		return fmt.Errorf("starting manager: %w", err)
	}

	if flags.bulwardSystemNamespace == "" {
		return fmt.Errorf("-bulward-system-namespace or ENVVAR BULWARD_NAMESPACE must be set")
	}

	if err = (&controllers.OrganizationReconciler{
		Client: mgr.GetClient(),
		Log:    log.WithName("controllers").WithName("Organization"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("creating Organization controller: %w", err)
	}

	if err = (&controllers.OrganizationRoleTemplateReconciler{
		Client: mgr.GetClient(),
		Log:    log.WithName("controllers").WithName("OrganizationRoleTemplate"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("creating OrganizationRoleTemplate controller: %w", err)
	}

	if err = (&controllers.ProjectReconciler{
		Client: mgr.GetClient(),
		Log:    log.WithName("controllers").WithName("Project"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("creating Project controller: %w", err)
	}

	if err = (&controllers.ProjectRoleTemplateReconciler{
		Client: mgr.GetClient(),
		Log:    log.WithName("controllers").WithName("ProjectRoleTemplate"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("creating ProjectRoleTemplate controller: %w", err)
	}

	if err := mgr.AddReadyzCheck("ping", healthz.Ping); err != nil {
		return fmt.Errorf("adding readyz checker: %w", err)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		return fmt.Errorf("adding healthz checker: %w", err)
	}

	log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("running manager: %w", err)
	}
	return nil
}
