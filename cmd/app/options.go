package app

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cliflag "k8s.io/component-base/cli/flag"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client"
	"github.com/jetstack/version-checker/pkg/client/selfhosted"
)

const (
	envPrefix = "VERSION_CHECKER"

	envACRUsername     = "ACR_USERNAME"
	envACRPassword     = "ACR_PASSWORD"
	envACRRefreshToken = "ACR_REFRESH_TOKEN"

	envDockerUsername = "DOCKER_USERNAME"
	envDockerPassword = "DOCKER_PASSWORD"
	envDockerToken    = "DOCKER_TOKEN"

	envECRIamRoleArn      = "ECR_IAM_ROLE_ARN"
	envECRAccessKeyID     = "ECR_ACCESS_KEY_ID"
	envECRSecretAccessKey = "ECR_SECRET_ACCESS_KEY"
	envECRSessionToken    = "ECR_SESSION_TOKEN"

	envGCRAccessToken = "GCR_TOKEN"

	envGHCRAccessToken = "GHCR_TOKEN"

	envQuayToken = "QUAY_TOKEN"

	envSelfhostedPrefix    = "SELFHOSTED"
	envSelfhostedUsername  = "USERNAME"
	envSelfhostedPassword  = "PASSWORD"
	envSelfhostedHost      = "HOST"
	envSelfhostedBearer    = "TOKEN"
	envSelfhostedTokenPath = "TOKEN_PATH"
	envSelfhostedInsecure  = "INSECURE"
	envSelfhostedCAPath    = "CA_PATH"
)

var (
	selfhostedHostReg     = regexp.MustCompile("^VERSION_CHECKER_SELFHOSTED_HOST_(.*)")
	selfhostedUsernameReg = regexp.MustCompile("^VERSION_CHECKER_SELFHOSTED_USERNAME_(.*)")
	selfhostedPasswordReg = regexp.MustCompile("^VERSION_CHECKER_SELFHOSTED_PASSWORD_(.*)")
	selfhostedTokenPath   = regexp.MustCompile("^VERSION_CHECKER_SELFHOSTED_TOKEN_PATH_(.*)")
	selfhostedTokenReg    = regexp.MustCompile("^VERSION_CHECKER_SELFHOSTED_TOKEN_(.*)")
	selfhostedCAPath      = regexp.MustCompile("^VERSION_CHECKER_SELFHOSTED_CA_PATH_(.*)")
	selfhostedInsecureReg = regexp.MustCompile("^VERSION_CHECKER_SELFHOSTED_INSECURE_(.*)")
)

// Options is a struct to hold options for the version-checker
type Options struct {
	MetricsServingAddress string
	DefaultTestAll        bool
	CacheTimeout          time.Duration
	LogLevel              string

	kubeConfigFlags *genericclioptions.ConfigFlags
	selfhosted      selfhosted.Options

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
			fmt.Sprintf(`annotation "%s/${my-container}=false".`, api.EnableAnnotationKey))

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
			envPrefix, envDockerToken,
		))
	///

	/// ECR
	fs.StringVar(&o.Client.ECR.IamRoleArn,
		"ecr-iam-role-arn", "",
		fmt.Sprintf(
			"IAM role ARN for read access to private registries, can not be used with access-key/secret-key/session-token (%s_%s).",
			envPrefix, envECRIamRoleArn,
		))
	fs.StringVar(&o.Client.ECR.AccessKeyID,
		"ecr-access-key-id", "",
		fmt.Sprintf(
			"ECR access key ID for read access to private registries (%s_%s).",
			envPrefix, envECRAccessKeyID,
		))
	fs.StringVar(&o.Client.ECR.SecretAccessKey,
		"ecr-secret-access-key", "",
		fmt.Sprintf(
			"ECR secret access key for read access to private registries (%s_%s).",
			envPrefix, envECRSecretAccessKey,
		))
	fs.StringVar(&o.Client.ECR.SessionToken,
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

	/// GHCR
	fs.StringVar(&o.Client.GHCR.Token,
		"gchr-token", "",
		fmt.Sprintf(
			"Personal Access token for read access to GHCR releases (%s_%s).",
			envPrefix, envGHCRAccessToken,
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
	fs.StringVar(&o.selfhosted.Username,
		"selfhosted-username", "",
		fmt.Sprintf(
			"Username is authenticate with a selfhosted registry (%s_%s).",
			envPrefix, envSelfhostedUsername,
		))
	fs.StringVar(&o.selfhosted.Password,
		"selfhosted-password", "",
		fmt.Sprintf(
			"Password is authenticate with a selfhosted registry (%s_%s).",
			envPrefix, envSelfhostedPassword,
		))
	fs.StringVar(&o.selfhosted.Bearer,
		"selfhosted-token", "",
		fmt.Sprintf(
			"Token to authenticate to a selfhosted registry. Cannot be used with "+
				"username/password (%s_%s).",
			envPrefix, envSelfhostedBearer,
		))
	fs.StringVar(&o.selfhosted.TokenPath,
		"selfhosted-token", "",
		fmt.Sprintf(
			"Override the default selfhosted registry's token auth path. "+
				"(%s_%s).",
			envPrefix, envSelfhostedTokenPath,
		))
	fs.StringVar(&o.selfhosted.Host,
		"selfhosted-registry-host", "",
		fmt.Sprintf(
			"Full host of the selfhosted registry. Include http[s] scheme (%s_%s)",
			envPrefix, envSelfhostedHost,
		))
	fs.StringVar(&o.selfhosted.Host,
		"selfhosted-registry-ca-path", "",
		fmt.Sprintf(
			"Absolute path to a PEM encoded x509 certificate chain. (%s_%s)",
			envPrefix, envSelfhostedCAPath,
		))
	fs.BoolVarP(&o.selfhosted.Insecure,
		"selfhosted-insecure", "", false,
		fmt.Sprintf(
			"Enable/Disable SSL Certificate Validation. WARNING: "+
				"THIS IS NOT RECOMMENDED AND IS INTENDED FOR DEBUGGING (%s_%s)",
			envPrefix, envSelfhostedInsecure,
		))
	///
}

func (o *Options) complete() {
	o.Client.Selfhosted = make(map[string]*selfhosted.Options)

	envs := os.Environ()
	for _, opt := range []struct {
		key    string
		assign *string
	}{
		{envACRUsername, &o.Client.ACR.Username},
		{envACRPassword, &o.Client.ACR.Password},
		{envACRRefreshToken, &o.Client.ACR.RefreshToken},

		{envDockerUsername, &o.Client.Docker.Username},
		{envDockerPassword, &o.Client.Docker.Password},
		{envDockerToken, &o.Client.Docker.Token},

		{envECRIamRoleArn, &o.Client.ECR.IamRoleArn},
		{envECRAccessKeyID, &o.Client.ECR.AccessKeyID},
		{envECRSessionToken, &o.Client.ECR.SessionToken},
		{envECRSecretAccessKey, &o.Client.ECR.SecretAccessKey},

		{envGCRAccessToken, &o.Client.GCR.Token},

		{envGHCRAccessToken, &o.Client.GHCR.Token},

		{envQuayToken, &o.Client.Quay.Token},
	} {
		for _, env := range envs {
			if o.assignEnv(env, opt.key, opt.assign) {
				break
			}
		}
	}

	o.assignSelfhosted(envs)
}

func (o *Options) assignEnv(env, key string, assign *string) bool {
	pair := strings.SplitN(env, "=", 2)
	if len(pair) < 2 {
		return false
	}

	if envPrefix+"_"+key == pair[0] && len(*assign) == 0 {
		*assign = pair[1]
		return true
	}

	return false
}

func (o *Options) assignSelfhosted(envs []string) {
	if o.Client.Selfhosted == nil {
		o.Client.Selfhosted = make(map[string]*selfhosted.Options)
	}

	initOptions := func(name string) {
		if o.Client.Selfhosted[name] == nil {
			o.Client.Selfhosted[name] = new(selfhosted.Options)
		}
	}

	for _, env := range envs {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 || len(pair[1]) == 0 {
			continue
		}

		if matches := selfhostedHostReg.FindStringSubmatch(strings.ToUpper(pair[0])); len(matches) == 2 {
			initOptions(matches[1])
			o.Client.Selfhosted[matches[1]].Host = pair[1]
			continue
		}

		if matches := selfhostedUsernameReg.FindStringSubmatch(strings.ToUpper(pair[0])); len(matches) == 2 {
			initOptions(matches[1])
			o.Client.Selfhosted[matches[1]].Username = pair[1]
			continue
		}

		if matches := selfhostedPasswordReg.FindStringSubmatch(strings.ToUpper(pair[0])); len(matches) == 2 {
			initOptions(matches[1])
			o.Client.Selfhosted[matches[1]].Password = pair[1]
			continue
		}

		if matches := selfhostedTokenPath.FindStringSubmatch(strings.ToUpper(pair[0])); len(matches) == 2 {
			initOptions(matches[1])
			o.Client.Selfhosted[matches[1]].TokenPath = pair[1]
			continue
		}

		if matches := selfhostedTokenReg.FindStringSubmatch(strings.ToUpper(pair[0])); len(matches) == 2 {
			initOptions(matches[1])
			o.Client.Selfhosted[matches[1]].Bearer = pair[1]
			continue
		}

		if matches := selfhostedInsecureReg.FindStringSubmatch(strings.ToUpper(pair[0])); len(matches) == 2 {
			initOptions(matches[1])
			val, err := strconv.ParseBool(pair[1])
			if err == nil {
				o.Client.Selfhosted[matches[1]].Insecure = val
			}
			continue
		}

		if matches := selfhostedCAPath.FindStringSubmatch(strings.ToUpper(pair[0])); len(matches) == 2 {
			initOptions(matches[1])
			o.Client.Selfhosted[matches[1]].CAPath = pair[1]
			continue
		}
	}

	if len(o.selfhosted.Host) > 0 {
		o.Client.Selfhosted[o.selfhosted.Host] = &o.selfhosted
	}
}
