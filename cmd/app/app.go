package app

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Load all auth plugins

	"github.com/jetstack/version-checker/pkg/client"
	"github.com/jetstack/version-checker/pkg/controller"
	"github.com/jetstack/version-checker/pkg/metrics"
)

const (
	helpOutput = "Kubernetes utility for exposing used image versions compared to the latest version, as metrics."

	envPrefix         = "VERSION_CHECKER"
	envGCRAccessToken = "GCR_TOKEN"
	envDockerUsername = "DOCKER_USERNAME"
	envDockerPassword = "DOCKER_PASSWORD"
	envDockerJWT      = "DOCKER_TOKEN"
	envQuayToken      = "QUAY_TOKEN"
)

// Options is a struct to hold options for the version-checker
type Options struct {
	MetricsServingAddress string
	DefaultTestAll        bool
	CacheTimeout          time.Duration
	LogLevel              string

	Client client.Options
}

func NewCommand(ctx context.Context) *cobra.Command {
	opts := new(Options)
	kubeConfigFlags := genericclioptions.NewConfigFlags(true)

	cmd := &cobra.Command{
		Use:   "version-checker",
		Short: helpOutput,
		Long:  helpOutput,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.checkEnv()

			logLevel, err := logrus.ParseLevel(opts.LogLevel)
			if err != nil {
				return fmt.Errorf("failed to parse --log-level %q: %s",
					opts.LogLevel, err)
			}

			nlog := logrus.New()
			nlog.SetOutput(os.Stdout)
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

			client, err := client.New(ctx, opts.Client)
			if err != nil {
				return fmt.Errorf("failed to setup image registry clients: %s", err)
			}

			defer func() {
				if err := metrics.Shutdown(); err != nil {
					log.Error(err)
				}
			}()

			c := controller.New(opts.CacheTimeout, metrics,
				client, kubeClient, log, opts.DefaultTestAll)

			return c.Run(ctx, opts.CacheTimeout/2)
		},
	}

	kubeConfigFlags.AddFlags(cmd.PersistentFlags())
	opts.addFlags(cmd)

	return cmd
}

func (o *Options) addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&o.MetricsServingAddress,
		"metrics-serving-address", "m", "0.0.0.0:8080",
		"Address to serve metrics on at the /metrics path.")

	cmd.PersistentFlags().BoolVarP(&o.DefaultTestAll,
		"test-all-containers", "a", false,
		`If enable, all containers will be tested, unless they have the annotation `+
			`"enable.version-checker/${my-container}=false".`)

	cmd.PersistentFlags().DurationVarP(&o.CacheTimeout,
		"image-cache-timeout", "c", time.Minute*30,
		"The time for an image in the cache to be considered fresh. Images will be "+
			"checked at this interval.")

	cmd.PersistentFlags().StringVarP(&o.LogLevel,
		"log-level", "v", "info",
		"Log level (debug, info, warn, error, fatal, panic).")

	cmd.PersistentFlags().StringVar(&o.Client.GCR.Token,
		"gcr-token", "",
		fmt.Sprintf(
			"Access token for read access to private GCR registries (%s_%s).",
			envPrefix, envGCRAccessToken,
		))

	cmd.PersistentFlags().StringVar(&o.Client.Quay.Token,
		"quay-token", "",
		fmt.Sprintf(
			"Access token for read access to private Quay registries (%s_%s).",
			envPrefix, envQuayToken,
		))

	cmd.PersistentFlags().StringVar(&o.Client.Docker.Username,
		"docker-username", "",
		fmt.Sprintf(
			"Username is authenticate with docker registry (%s_%s).",
			envPrefix, envDockerUsername,
		))
	cmd.PersistentFlags().StringVar(&o.Client.Docker.Password,
		"docker-password", "",
		fmt.Sprintf(
			"Password is authenticate with docker registry (%s_%s).",
			envPrefix, envDockerPassword,
		))
	cmd.PersistentFlags().StringVar(&o.Client.Docker.JWT,
		"docker-token", "",
		fmt.Sprintf(
			"Token is authenticate with docker registry. Cannot be used with "+
				"username/password (%s_%s).",
			envPrefix, envDockerJWT,
		))
	cmd.PersistentFlags().StringVar(&o.Client.Docker.LoginURL,
		"docker-login-url", "https://hub.docker.com/v2/users/login/",
		"URL to login into docker using username/password.")
}

func (o *Options) checkEnv() {
	if len(o.Client.GCR.Token) == 0 {
		o.Client.GCR.Token = os.Getenv(envPrefix + "_" + envGCRAccessToken)
	}

	if len(o.Client.Docker.Username) == 0 {
		o.Client.Docker.Username = os.Getenv(envPrefix + "_" + envDockerUsername)
	}
	if len(o.Client.Docker.Password) == 0 {
		o.Client.Docker.Password = os.Getenv(envPrefix + "_" + envDockerPassword)
	}
	if len(o.Client.Docker.JWT) == 0 {
		o.Client.Docker.JWT = os.Getenv(envPrefix + "_" + envDockerJWT)
	}

	if len(o.Client.Quay.Token) == 0 {
		o.Client.Quay.Token = os.Getenv(envPrefix + "_" + envQuayToken)
	}
}
