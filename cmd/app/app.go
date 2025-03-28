package app

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/go-chi/transport"
	"github.com/hashicorp/go-cleanhttp"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Load all auth plugins

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client"
	"github.com/jetstack/version-checker/pkg/controller"
	"github.com/jetstack/version-checker/pkg/metrics"
)

const (
	helpOutput = "Kubernetes utility for exposing used image versions compared to the latest version, as metrics."
)

func NewCommand(ctx context.Context) *cobra.Command {
	opts := new(Options)

	cmd := &cobra.Command{
		Use:   "version-checker",
		Short: helpOutput,
		Long:  helpOutput,
		RunE: func(_ *cobra.Command, _ []string) error {
			opts.complete()

			logLevel, err := logrus.ParseLevel(opts.LogLevel)
			if err != nil {
				return fmt.Errorf("failed to parse --log-level %q: %s",
					opts.LogLevel, err)
			}
			log := newLogger(logLevel)

			restConfig, err := opts.kubeConfigFlags.ToRESTConfig()
			if err != nil {
				return fmt.Errorf("failed to build kubernetes rest config: %s", err)
			}

			kubeClient, err := kubernetes.NewForConfig(restConfig)
			if err != nil {
				return fmt.Errorf("failed to build kubernetes client: %s", err)
			}

			metricsServer := metrics.NewServer(log)
			if err := metricsServer.Run(opts.MetricsServingAddress); err != nil {
				return fmt.Errorf("failed to start metrics server: %s", err)
			}

			opts.Client.Transport = transport.Chain(
				cleanhttp.DefaultTransport(),
				metricsServer.RoundTripper,
			)

			client, err := client.New(ctx, log, opts.Client)
			if err != nil {
				return fmt.Errorf("failed to setup image registry clients: %s", err)
			}

			defer func() {
				if err := metricsServer.Shutdown(); err != nil {
					log.Error(err)
				}
			}()

			defaultTestAllInfoMsg := fmt.Sprintf(`only containers with the annotation "%s/${my-container}=true" will be parsed`, api.EnableAnnotationKey)
			if opts.DefaultTestAll {
				defaultTestAllInfoMsg = fmt.Sprintf(`all containers will be tested, unless they have the annotation "%s/${my-container}=false"`, api.EnableAnnotationKey)
			}

			log.Infof("flag --test-all-containers=%t %s", opts.DefaultTestAll, defaultTestAllInfoMsg)

			c := controller.New(opts.CacheTimeout, metricsServer,
				client, kubeClient, log, opts.DefaultTestAll)

			return c.Run(ctx, opts.CacheTimeout/2)
		},
	}

	opts.addFlags(cmd)

	return cmd
}
