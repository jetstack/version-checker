package app

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cliflag "k8s.io/component-base/cli/flag"

	"github.com/jetstack/version-checker/pkg/client"
)

const (
	envPrefix = "VERSION_CHECKER"

	envACRUsername     = "ACR_USERNAME"
	envACRPassword     = "ACR_PASSWORD"
	envACRRefreshToken = "ACR_REFRESH_TOKEN"

	envECRAccessKeyID     = "ECR_ACCESS_KEY_ID"
	envECRSecretAccessKey = "ECR_SECRET_ACCESS_KEY"
	envECRSessionToken    = "ECR_SESSION_TOKEN"

	envDockerUsername = "DOCKER_USERNAME"
	envDockerPassword = "DOCKER_PASSWORD"
	envDockerJWT      = "DOCKER_TOKEN"

	envGCRAccessToken = "GCR_TOKEN"

	envQuayToken = "QUAY_TOKEN"

	envSelfhostedUsername = "SELFHOSTED_USERNAME"
	envSelfhostedPassword = "SELFHOSTED_PASSWORD"
	envSelfhostedBearer   = "SELFHOSTED_TOKEN"
)

// Options is a struct to hold options for the version-checker
type Options struct {
	MetricsServingAddress string
	DefaultTestAll        bool
	CacheTimeout          time.Duration
	LogLevel              string

	kubeConfigFlags *genericclioptions.ConfigFlags

	Client client.Options
}

func (o *Options) addFlags(cmd *cobra.Command) {
	var nfs cliflag.NamedFlagSets

	o.addAppFlags(nfs.FlagSet("App"))
	o.addAuthFlags(nfs.FlagSet("Auth"))
	o.kubeConfigFlags = genericclioptions.NewConfigFlags(true)
	o.kubeConfigFlags.AddFlags(nfs.FlagSet("Kubernetes"))

	usageFmt := "Usage:\n  %s\n"
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), nfs, 0)
		return nil
	})

	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), nfs, 0)
	})

	fs := cmd.Flags()
	for _, f := range nfs.FlagSets {
		fs.AddFlagSet(f)
	}
}

func (o *Options) addAppFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.MetricsServingAddress,
		"metrics-serving-address", "m", "0.0.0.0:8080",
		"Address to serve metrics on at the /metrics path.")

	fs.BoolVarP(&o.DefaultTestAll,
		"test-all-containers", "a", false,
		"If enabled, all containers will be tested, unless they have the "+
			`annotation "enable.version-checker/${my-container}=false".`)

	fs.DurationVarP(&o.CacheTimeout,
		"image-cache-timeout", "c", time.Minute*30,
		"The time for an image version in the cache to be considered fresh. Images "+
			"will be rechecked after this interval.")

	fs.StringVarP(&o.LogLevel,
		"log-level", "v", "info",
		"Log level (debug, info, warn, error, fatal, panic).")
}

func (o *Options) addAuthFlags(fs *pflag.FlagSet) {
	/// ACR
	fs.StringVar(&o.Client.ACR.Username,
		"acr-username", "",
		fmt.Sprintf(
			"Username to authenticate with azure container registry (%s_%s).",
			envPrefix, envACRUsername,
		))
	fs.StringVar(&o.Client.ACR.Password,
		"acr-password", "",
		fmt.Sprintf(
			"Password to authenticate with azure container registry (%s_%s).",
			envPrefix, envACRPassword,
		))
	fs.StringVar(&o.Client.ACR.RefreshToken,
		"acr-refresh-token", "",
		fmt.Sprintf(
			"Refresh token to authenticate with azure container registry. Cannot be used with "+
				"username/password (%s_%s).",
			envPrefix, envACRRefreshToken,
		))
	///

	// Docker
	fs.StringVar(&o.Client.Docker.Username,
		"docker-username", "",
		fmt.Sprintf(
			"Username to authenticate with docker registry (%s_%s).",
			envPrefix, envDockerUsername,
		))
	fs.StringVar(&o.Client.Docker.Password,
		"docker-password", "",
		fmt.Sprintf(
			"Password to authenticate with docker registry (%s_%s).",
			envPrefix, envDockerPassword,
		))
	fs.StringVar(&o.Client.Docker.Token,
		"docker-token", "",
		fmt.Sprintf(
			"Token to authenticate with docker registry. Cannot be used with "+
				"username/password (%s_%s).",
			envPrefix, envDockerJWT,
		))
	///

	/// ECR
	fs.StringVar(&o.Client.GCR.Token,
		"ecr-access-key-id", "",
		fmt.Sprintf(
			"ECR access key ID for read access to private registries (%s_%s).",
			envPrefix, envECRAccessKeyID,
		))
	fs.StringVar(&o.Client.GCR.Token,
		"ecr-secret-access-key", "",
		fmt.Sprintf(
			"ECR secret access key for read access to private registries (%s_%s).",
			envPrefix, envECRSecretAccessKey,
		))
	fs.StringVar(&o.Client.GCR.Token,
		"ecr-session-token", "",
		fmt.Sprintf(
			"ECR session token for read access to private registries (%s_%s).",
			envPrefix, envECRSessionToken,
		))
	///

	/// GCR
	fs.StringVar(&o.Client.GCR.Token,
		"gcr-token", "",
		fmt.Sprintf(
			"Access token for read access to private GCR registries (%s_%s).",
			envPrefix, envGCRAccessToken,
		))
	///

	/// Quay
	fs.StringVar(&o.Client.Quay.Token,
		"quay-token", "",
		fmt.Sprintf(
			"Access token for read access to private Quay registries (%s_%s).",
			envPrefix, envQuayToken,
		))
	///

	/// Selfhosted
	fs.StringVar(&o.Client.Selfhosted.Username,
		"selfhosted-username", "",
		fmt.Sprintf(
			"Username is authenticate with a selfhosted registry (%s_%s).",
			envPrefix, envSelfhostedUsername,
		))
	fs.StringVar(&o.Client.Selfhosted.Password,
		"selfhosted-password", "",
		fmt.Sprintf(
			"Password is authenticate with a selfhosted registry (%s_%s).",
			envPrefix, envSelfhostedPassword,
		))
	fs.StringVar(&o.Client.Selfhosted.Bearer,
		"selfhosted-token", "",
		fmt.Sprintf(
			"Token to authenticate to a selfhosted registry. Cannot be used with "+
				"username/password (%s_%s).",
			envPrefix, envSelfhostedBearer,
		))
	fs.StringVar(&o.Client.Selfhosted.URL,
		"selfhosted-registry-url", "",
		"URL of the selfhosted registry.")
	///
}

func (o *Options) complete() {
	// ACR
	if len(o.Client.ACR.Username) == 0 {
		o.Client.ACR.Username = os.Getenv(envPrefix + "_" + envACRUsername)
	}
	if len(o.Client.ACR.Password) == 0 {
		o.Client.ACR.Password = os.Getenv(envPrefix + "_" + envACRPassword)
	}
	if len(o.Client.ACR.RefreshToken) == 0 {
		o.Client.ACR.RefreshToken = os.Getenv(envPrefix + "_" + envACRRefreshToken)
	}

	// Docker
	if len(o.Client.Docker.Username) == 0 {
		o.Client.Docker.Username = os.Getenv(envPrefix + "_" + envDockerUsername)
	}
	if len(o.Client.Docker.Password) == 0 {
		o.Client.Docker.Password = os.Getenv(envPrefix + "_" + envDockerPassword)
	}
	if len(o.Client.Docker.Token) == 0 {
		o.Client.Docker.Token = os.Getenv(envPrefix + "_" + envDockerJWT)
	}

	// ECR
	if len(o.Client.ECR.AccessKeyID) == 0 {
		o.Client.ECR.AccessKeyID = os.Getenv(envPrefix + "_" + envECRAccessKeyID)
	}
	if len(o.Client.ECR.SecretAccessKey) == 0 {
		o.Client.ECR.SecretAccessKey = os.Getenv(envPrefix + "_" + envECRSecretAccessKey)
	}
	if len(o.Client.ECR.SessionToken) == 0 {
		o.Client.ECR.SessionToken = os.Getenv(envPrefix + "_" + envECRSessionToken)
	}

	// GCR
	if len(o.Client.GCR.Token) == 0 {
		o.Client.GCR.Token = os.Getenv(envPrefix + "_" + envGCRAccessToken)
	}

	// Quay
	if len(o.Client.Quay.Token) == 0 {
		o.Client.Quay.Token = os.Getenv(envPrefix + "_" + envQuayToken)
	}

	// Quay
	if len(o.Client.Quay.Token) == 0 {
		o.Client.Quay.Token = os.Getenv(envPrefix + "_" + envQuayToken)
	}

	// Selfhosted
	if len(o.Client.Selfhosted.Username) == 0 {
		o.Client.Selfhosted.Username = os.Getenv(envPrefix + "_" + envSelfhostedUsername)
	}
	if len(o.Client.Selfhosted.Password) == 0 {
		o.Client.Selfhosted.Password = os.Getenv(envPrefix + "_" + envSelfhostedPassword)
	}
	if len(o.Client.Selfhosted.Bearer) == 0 {
		o.Client.Selfhosted.Bearer = os.Getenv(envPrefix + "_" + envSelfhostedBearer)
	}
}
