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
)

const (
	envPrefix = "VERSION_CHECKER"

	envACRUsername     = "ACR_USERNAME"
	envACRPassword     = "ACR_PASSWORD"      // #nosec G101
	envACRRefreshToken = "ACR_REFRESH_TOKEN" // #nosec G101
	envACRJWKSURI      = "ACR_JWKS_URI"

	envDockerUsername = "DOCKER_USERNAME"
	envDockerPassword = "DOCKER_PASSWORD" // #nosec G101
	envDockerToken    = "DOCKER_TOKEN"    // #nosec G101

	envECRIamRoleArn      = "ECR_IAM_ROLE_ARN"
	envECRAccessKeyID     = "ECR_ACCESS_KEY_ID"     // #nosec G101
	envECRSecretAccessKey = "ECR_SECRET_ACCESS_KEY" // #nosec G101
	envECRSessionToken    = "ECR_SESSION_TOKEN"     // #nosec G101

	envGCRAccessToken = "GCR_TOKEN" // #nosec G101

	envGHCRAccessToken = "GHCR_TOKEN" // #nosec G101
	envGHCRHostname    = "GHCR_HOSTNAME"

	envQuayToken = "QUAY_TOKEN" // #nosec G101

	// Used for kubernetes Credential Discovery
	envKeychainServiceAccountName = "AUTH_SERVICE_ACCOUNT_NAME"
	envKeychainNamespace          = "AUTH_SERVICE_ACCOUNT_NAMESPACE"
	envKeychainImagePullSecrets   = "AUTH_IMAGE_PULL_SECRETS"
	envKeychainUseMountSecrets    = "AUTH_USE_MOUNT_SECRETS"
	// Duration in which to Refresh Credentials from Service Account
	envKeychainRefreshDuration = "AUTH_REFRESH_DURATION"
)

// Options is a struct to hold options for the version-checker.
type Options struct {
	MetricsServingAddress string
	PprofBindAddress      string

	DefaultTestAll bool
	LogLevel       string

	CacheTimeout            time.Duration
	GracefulShutdownTimeout time.Duration
	CacheSyncPeriod         time.Duration
	RequeueDuration         time.Duration

	kubeConfigFlags *genericclioptions.ConfigFlags

	Client client.Options
}

type envMatcher struct {
	re     *regexp.Regexp
	action func(matches []string, value string)
}

func (o *Options) addFlags(cmd *cobra.Command) {
	var nfs cliflag.NamedFlagSets

	o.addAppFlags(nfs.FlagSet("App"))
	o.addAuthFlags(nfs.FlagSet("Auth"))
	o.kubeConfigFlags = genericclioptions.NewConfigFlags(true)
	o.kubeConfigFlags.AddFlags(nfs.FlagSet("Kubernetes"))

	usageFmt := "Usage:\n  %s\n"
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		_, _ = fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), nfs, 0)
		return nil
	})

	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
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

	fs.StringVarP(&o.PprofBindAddress,
		"pprof-serving-address", "", "",
		"Address to serve pprof on for profiling.")

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

	fs.DurationVarP(&o.GracefulShutdownTimeout,
		"graceful-shutdown-timeout", "", 10*time.Second,
		"Time that the manager should wait for all controller to shutdown.")

	fs.DurationVarP(&o.RequeueDuration,
		"requeue-duration", "r", time.Hour,
		"The time a pod will be re-checked for new versions/tags")

	fs.DurationVarP(&o.CacheSyncPeriod,
		"cache-sync-period", "", 5*time.Hour,
		"The time in which all resources should be updated.")
}

func (o *Options) addAuthFlags(fs *pflag.FlagSet) {

	/// KEYCHAIN
	fs.StringVar(&o.Client.KeyChain.Namespace,
		"keychain-namespace", "",
		fmt.Sprintf(
			"Namespace inside of which service account and imagepullsecrets belong too (%s_%s).",
			envPrefix, envKeychainNamespace,
		))

	fs.StringVar(&o.Client.KeyChain.ServiceAccountName,
		"keychain-service-account", "",
		fmt.Sprintf(
			"ServiceAccount used to fetch Image Pull Secrets from (%s_%s).",
			envPrefix, envKeychainServiceAccountName,
		))

	fs.StringSliceVar(&o.Client.KeyChain.ImagePullSecrets,
		"keychain-image-pull-secrets", []string{},
		fmt.Sprintf(
			"Set of image pull secrets to include during authentication (%s_%s).",
			envPrefix, envKeychainImagePullSecrets,
		))

	fs.BoolVar(&o.Client.KeyChain.UseMountSecrets,
		"keychain-use-mount-secrets", false,
		fmt.Sprintf("Include Mount Secrets during discovery (%s_%s).",
			envPrefix, envKeychainUseMountSecrets,
		))
	fs.DurationVar(&o.Client.AuthRefreshDuration,
		"keychain-refresh-duration", time.Hour,
		fmt.Sprintf("Duration credentials are refreshed (%s_%s).",
			envPrefix, envKeychainRefreshDuration,
		))

	/// ACR
	fs.StringVar(&o.Client.ACR.Username,
		"acr-username", "",
		fmt.Sprintf(
			"Username to authenticate with azure container registry (%s_%s).",
			envPrefix, envACRUsername,
		))
	_ = fs.MarkDeprecated("acr-username", "use keychain instead")
	fs.StringVar(&o.Client.ACR.Password,
		"acr-password", "",
		fmt.Sprintf(
			"Password to authenticate with azure container registry (%s_%s).",
			envPrefix, envACRPassword,
		))
	_ = fs.MarkDeprecated("acr-password", "use keychain instead")
	fs.StringVar(&o.Client.ACR.RefreshToken,
		"acr-refresh-token", "",
		fmt.Sprintf(
			"Refresh token to authenticate with azure container registry. Cannot be used with "+
				"username/password (%s_%s).",
			envPrefix, envACRRefreshToken,
		))
	_ = fs.MarkDeprecated("acr-refresh-token", "use keychain instead")
	fs.StringVar(&o.Client.ACR.JWKSURI,
		"acr-jwks-uri", "",
		fmt.Sprintf(
			"JWKS URI to verify the JWT access token received. If left blank, JWT token will not be verified. (%s_%s)",
			envPrefix, envACRJWKSURI,
		))
	///

	// Docker
	fs.StringVar(&o.Client.Docker.Username,
		"docker-username", "",
		fmt.Sprintf(
			"Username to authenticate with docker registry (%s_%s).",
			envPrefix, envDockerUsername,
		))
	_ = fs.MarkDeprecated("docker-username", "use keychain instead")
	fs.StringVar(&o.Client.Docker.Password,
		"docker-password", "",
		fmt.Sprintf(
			"Password to authenticate with docker registry (%s_%s).",
			envPrefix, envDockerPassword,
		))
	_ = fs.MarkDeprecated("docker-password", "use keychain instead")
	fs.StringVar(&o.Client.Docker.Token,
		"docker-token", "",
		fmt.Sprintf(
			"Token to authenticate with docker registry. Cannot be used with "+
				"username/password (%s_%s).",
			envPrefix, envDockerToken,
		))
	_ = fs.MarkDeprecated("docker-token", "use keychain instead")
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
	_ = fs.MarkDeprecated("gcr-token", "use keychain instead")
	///

	/// GHCR
	fs.StringVar(&o.Client.GHCR.Token,
		"gchr-token", "",
		fmt.Sprintf(
			"Personal Access token for read access to GHCR releases (%s_%s).",
			envPrefix, envGHCRAccessToken,
		))
	_ = fs.MarkDeprecated("gchr-token", "use keychain instead")
	fs.StringVar(&o.Client.GHCR.Hostname,
		"gchr-hostname", "",
		fmt.Sprintf(
			"Override hostname for Github Enterprise instances (%s_%s).",
			envPrefix, envGHCRHostname,
		))
	///

	/// Quay
	fs.StringVar(&o.Client.Quay.Token,
		"quay-token", "",
		fmt.Sprintf(
			"Access token for read access to private Quay registries (%s_%s).",
			envPrefix, envQuayToken,
		))
	_ = fs.MarkDeprecated("quay-token", "use keychain instead")
	///

	/// Selfhosted
	fs.StringVar(&o.selfhosted.Username,
		"selfhosted-username", "",
		fmt.Sprintf(
			"Username is authenticate with a selfhosted registry (%s_%s_%s).",
			envPrefix, envSelfhostedPrefix, envSelfhostedUsername,
		))
	_ = fs.MarkDeprecated("selfhosted-username", "use keychain instead")
	fs.StringVar(&o.selfhosted.Password,
		"selfhosted-password", "",
		fmt.Sprintf(
			"Password is authenticate with a selfhosted registry (%s_%s_%s).",
			envPrefix, envSelfhostedPrefix, envSelfhostedPassword,
		))
	_ = fs.MarkDeprecated("selfhosted-password", "use keychain instead")
	fs.StringVar(&o.selfhosted.Bearer,
		"selfhosted-token", "",
		fmt.Sprintf(
			"Token to authenticate to a selfhosted registry. Cannot be used with "+
				"username/password (%s_%s_%s).",
			envPrefix, envSelfhostedPrefix, envSelfhostedBearer,
		))
	_ = fs.MarkDeprecated("selfhosted-token", "use keychain instead")
	fs.StringVar(&o.selfhosted.TokenPath,
		"selfhosted-token-path", "",
		fmt.Sprintf(
			"Override the default selfhosted registry's token auth path. "+
				"(%s_%s_%s).",
			envPrefix, envSelfhostedPrefix, envSelfhostedTokenPath,
		))
	fs.StringVar(&o.selfhosted.Host,
		"selfhosted-registry-host", "",
		fmt.Sprintf(
			"Full host of the selfhosted registry. Include http[s] scheme (%s_%s_%s)",
			envPrefix, envSelfhostedPrefix, envSelfhostedHost,
		))
	fs.StringVar(&o.selfhosted.CAPath,
		"selfhosted-registry-ca-path", "",
		fmt.Sprintf(
			"Absolute path to a PEM encoded x509 certificate chain. (%s_%s_%s)",
			envPrefix, envSelfhostedPrefix, envSelfhostedCAPath,
		))
	fs.BoolVarP(&o.selfhosted.Insecure,
		"selfhosted-insecure", "", false,
		fmt.Sprintf(
			"Enable/Disable SSL Certificate Validation. WARNING: "+
				"THIS IS NOT RECOMMENDED AND IS INTENDED FOR DEBUGGING (%s_%s_%s)",
			envPrefix, envSelfhostedPrefix, envSelfhostedInsecure,
		))
	fs.MarkDeprecated("selfhosted-insecure", "No longer supported, you MUST provide the CA Chain.")
}

func (o *Options) complete() error {
	o.Client.Selfhosted = make(map[string]*selfhosted.Options)

	envs := os.Environ()
	for _, opt := range []struct {
		key    string
		assign *string
	}{
		{envACRUsername, &o.Client.ACR.Username},
		{envACRPassword, &o.Client.ACR.Password},
		{envACRRefreshToken, &o.Client.ACR.RefreshToken},
		{envACRJWKSURI, &o.Client.ACR.JWKSURI},

		{envDockerUsername, &o.Client.Docker.Username},
		{envDockerPassword, &o.Client.Docker.Password},
		{envDockerToken, &o.Client.Docker.Token},

		{envECRIamRoleArn, &o.Client.ECR.IamRoleArn},
		{envECRAccessKeyID, &o.Client.ECR.AccessKeyID},
		{envECRSessionToken, &o.Client.ECR.SessionToken},
		{envECRSecretAccessKey, &o.Client.ECR.SecretAccessKey},

		{envGCRAccessToken, &o.Client.GCR.Token},

		{envGHCRAccessToken, &o.Client.GHCR.Token},
		{envGHCRHostname, &o.Client.GHCR.Hostname},

		{envQuayToken, &o.Client.Quay.Token},

		{envKeychainNamespace, &o.Client.KeyChain.Namespace},
		{envKeychainServiceAccountName, &o.Client.KeyChain.ServiceAccountName},
	} {
		for _, env := range envs {
			if o.assignEnv(env, opt.key, opt.assign) {
				break
			}
		}
	}

	return o.assignSelfhosted(envs)
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

// assignSelfhosted processes a list of environment variables and assigns
// self-hosted configuration options to the Options struct. It parses the
// environment variables using predefined regular expressions to extract
// self-hosted configuration details such as token path, bearer token, host,
// username, password, insecure flag, and CA path.
//
// The function ensures that each self-hosted configuration is initialized
// before assigning values. It also validates the self-hosted options after
// processing all environment variables.
//
// Parameters:
//   - envs: A slice of strings representing environment variables in the
//     format "KEY=VALUE".
//
// Returns:
//   - error: An error if validation of the self-hosted options fails, or nil
//     if the operation is successful.
func (o *Options) assignSelfhosted(envs []string) error {
	if o.Client.Selfhosted == nil {
		o.Client.Selfhosted = make(map[string]*selfhosted.Options)
	}

	initOptions := func(name string) {
		if o.Client.Selfhosted[name] == nil {
			o.Client.Selfhosted[name] = new(selfhosted.Options)
		}
	}

	// Go maps iterate in random order - Using a slice to consistency
	regexActions := []envMatcher{
		{
			re: selfhostedTokenPath,
			action: func(matches []string, value string) {
				initOptions(matches[1])
				o.Client.Selfhosted[matches[1]].TokenPath = value
			},
		},
		{
			re: selfhostedTokenReg,
			action: func(matches []string, value string) {
				initOptions(matches[1])
				o.Client.Selfhosted[matches[1]].Bearer = value
			},
		},
		// All your other patterns (host, username, password, insecure, capath...)
		{
			re: selfhostedHostReg,
			action: func(matches []string, value string) {
				initOptions(matches[1])
				o.Client.Selfhosted[matches[1]].Host = value
			},
		},
		{
			re: selfhostedUsernameReg,
			action: func(matches []string, value string) {
				initOptions(matches[1])
				o.Client.Selfhosted[matches[1]].Username = value
			},
		},
		{
			re: selfhostedPasswordReg,
			action: func(matches []string, value string) {
				initOptions(matches[1])
				o.Client.Selfhosted[matches[1]].Password = value
			},
		},
		{
			re: selfhostedInsecureReg,
			action: func(matches []string, value string) {
				initOptions(matches[1])
				if b, err := strconv.ParseBool(value); err == nil {
					o.Client.Selfhosted[matches[1]].Insecure = b
				}
			},
		},
		{
			re: selfhostedCAPath,
			action: func(matches []string, value string) {
				initOptions(matches[1])
				o.Client.Selfhosted[matches[1]].CAPath = value
			},
		},
	}

	for _, env := range envs {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 || parts[1] == "" {
			continue
		}
		key := strings.ToUpper(parts[0])
		val := parts[1]

		for _, p := range regexActions {
			if match := p.re.FindStringSubmatch(key); len(match) == 2 {
				p.action(match, val)
				break
			}
		}
	}

	// If we have some selfhosted flags, lets set them here...
	if len(o.selfhosted.Host) > 0 {
		o.Client.Selfhosted[o.selfhosted.Host] = &o.selfhosted
	}

	return validateSelfHostedOpts(o)
}

// validateSelfHostedOpts validates the self-hosted options provided in the
// Options struct. It checks both the options set using environment variables
// and those set using flags.
//
// For options set using environment variables, it iterates through the list
// of self-hosted options and ensures that each host is valid.
//
// For options set using flags, it validates the host in the selfhosted.Options
// struct.
//
// Returns an error if any of the self-hosted options contain an invalid host,
// otherwise returns nil.
func validateSelfHostedOpts(opts *Options) error {
	// opts set using env vars
	if opts.Client.Selfhosted != nil {
		for name, selfHostedOpts := range opts.Client.Selfhosted {
			if err := isValidOption(selfHostedOpts.Host, ""); !err {
				return fmt.Errorf("invalid self-hosted option for: %s", name)
			}
		}
	}

	// opts set using flags
	if opts.selfhosted != (selfhosted.Options{}) {
		if !isValidOption(opts.selfhosted.Host, "") {
			return fmt.Errorf("invalid self-hosted option for host: %s", opts.selfhosted.Host)
		}
	}
	return nil
}

func isValidOption(option, invalid any) bool {
	return option != invalid
}
