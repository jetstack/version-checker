package app

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Load all auth plugins

	"github.com/joshvanl/version-checker/pkg/controller"
	"github.com/joshvanl/version-checker/pkg/metrics"
)

const (
	version = "v0.0.1-alpha.0"

	helpOutput = "Kubernetes utility for exposing used image versions compared to the latest version, as metrics."
)

// Options is a struct to hold options for the version-checker
type Options struct {
	MetricsServingAddress string
	DefaultTestAll        bool
	CacheTimeout          time.Duration
	LogLevel              string
}

func NewCommand(ctx context.Context) *cobra.Command {
	opts := new(Options)
	kubeConfigFlags := genericclioptions.NewConfigFlags(true)

	cmd := &cobra.Command{
		Use:   "version-checker",
		Short: helpOutput,
		Long:  helpOutput,
		RunE: func(cmd *cobra.Command, args []string) error {
			logLevel, err := logrus.ParseLevel(opts.LogLevel)
			if err != nil {
				return fmt.Errorf("failed to parse --log-level %q: %s",
					opts.LogLevel, err)
			}

			nlog := logrus.New()
			nlog.SetLevel(logLevel)
			log := logrus.NewEntry(nlog)

			restConfig, err := kubeConfigFlags.ToRESTConfig()
			if err != nil {
				return fmt.Errorf("failed to build kubernetes rest config: %s", err)
			}

			kubeClient, err := kubernetes.NewForConfig(restConfig)
			if err != nil {
				return fmt.Errorf("failed to build kubernetes client: %s", err)
			}

			metrics := metrics.New(log)
			if err := metrics.Run(opts.MetricsServingAddress); err != nil {
				return fmt.Errorf("failed to start metrics server: %s", err)
			}

			defer func() {
				if err := metrics.Shutdown(); err != nil {
					log.Error(err)
				}
			}()

			c := controller.New(opts.CacheTimeout, metrics,
				kubeClient, log, opts.DefaultTestAll)
			return c.Run(ctx)
		},
	}

	kubeConfigFlags.AddFlags(cmd.PersistentFlags())

	cmd.PersistentFlags().StringVarP(&opts.MetricsServingAddress,
		"metrics-serving-address", "m", "0.0.0.0:8080",
		"Address to serve metrics on at the /metrics path.")

	cmd.PersistentFlags().BoolVarP(&opts.DefaultTestAll,
		"test-all-containers", "a", false,
		`If enable, all containers will be tested, unless they have the annotation `+
			`"enable.version-checker/${my-container}=false".`)

	cmd.PersistentFlags().DurationVarP(&opts.CacheTimeout,
		"image-cache-timeout", "c", time.Minute*30,
		"The time for an image in the cache to be considered fresh. Images will be "+
			"checked at this interval.")

	cmd.PersistentFlags().StringVarP(&opts.LogLevel,
		"log-level", "v", "info",
		"Log level (debug, info, warn, error, fatal, panic)")

	return cmd
}
