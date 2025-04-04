package app

import (
	"context"
	"fmt"
	"net/http"

	logrusr "github.com/bombsimon/logrusr/v4"
	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"

	"github.com/go-chi/transport"
	"github.com/hashicorp/go-cleanhttp"

	_ "k8s.io/client-go/plugin/pkg/client/auth" // Load all auth plugins

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client"
	"github.com/jetstack/version-checker/pkg/controller"
	"github.com/jetstack/version-checker/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
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

			log := newLogger(logLevel).WithField("component", "controller")
			ctrl.SetLogger(logrusr.New(log.WithField("controller", "manager").Logger))

			defaultTestAllInfoMsg := fmt.Sprintf(`only containers with the annotation "%s/${my-container}=true" will be parsed`, api.EnableAnnotationKey)
			if opts.DefaultTestAll {
				defaultTestAllInfoMsg = fmt.Sprintf(`all containers will be tested, unless they have the annotation "%s/${my-container}=false"`, api.EnableAnnotationKey)
			}

			restConfig, err := opts.kubeConfigFlags.ToRESTConfig()
			if err != nil {
				return fmt.Errorf("failed to build kubernetes rest config: %s", err)
			}

			log.Warnf("flag --test-all-containers=%t %s", opts.DefaultTestAll, defaultTestAllInfoMsg)

			mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
				LeaderElection: false,
				Metrics: server.Options{
					BindAddress:   opts.MetricsServingAddress,
					SecureServing: false,
				},
				GracefulShutdownTimeout: &opts.GracefulShutdownTimeout,
				Cache:                   cache.Options{SyncPeriod: &opts.CacheSyncPeriod},
				PprofBindAddress:        opts.PprofBindAddress,
			})
			if err != nil {
				return err
			}

			// Liveness probe
			if err := mgr.AddMetricsServerExtraHandler("/healthz",
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("ok"))
				})); err != nil {
				log.Fatal("Unable to set up health check:", err)
			}

			// Readiness probe
			if err := mgr.AddMetricsServerExtraHandler("/readyz",
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					if mgr.GetCache().WaitForCacheSync(context.Background()) {
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte("ready"))
					} else {
						http.Error(w, "cache not synced", http.StatusServiceUnavailable)
					}
				}),
			); err != nil {
				log.Fatal("Unable to set up ready check:", err)
			}

			metricsServer := metrics.New(log, ctrmetrics.Registry, mgr.GetCache())

			opts.Client.Transport = transport.Chain(
				cleanhttp.DefaultTransport(),
				metricsServer.RoundTripper,
			)

			client, err := client.New(ctx, log, opts.Client)
			if err != nil {
				return fmt.Errorf("failed to setup image registry clients: %s", err)
			}

			c := controller.NewPodReconciler(opts.CacheTimeout,
				metricsServer,
				client,
				mgr.GetClient(),
				log,
				opts.RequeueDuration,
				opts.DefaultTestAll,
			)

			if err := c.SetupWithManager(mgr); err != nil {
				return err
			}

			// Start the manager and all controllers
			log.Info("Starting controller manager")
			if err := mgr.Start(ctx); err != nil {
				return err
			}
			return nil
		},
	}

	opts.addFlags(cmd)

	return cmd
}
