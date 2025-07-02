package app

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jetstack/version-checker/pkg/client"
	"github.com/jetstack/version-checker/pkg/client/acr"
	"github.com/jetstack/version-checker/pkg/client/docker"
	"github.com/jetstack/version-checker/pkg/client/ecr"
	"github.com/jetstack/version-checker/pkg/client/gcr"
	"github.com/jetstack/version-checker/pkg/client/ghcr"
	"github.com/jetstack/version-checker/pkg/client/quay"
	"github.com/jetstack/version-checker/pkg/client/selfhosted"
)

func TestComplete(t *testing.T) {
	tests := map[string]struct {
		envs       [][2]string
		expOptions client.Options
	}{
		"no envs should give no options": {
			envs: [][2]string{},
			expOptions: client.Options{
				Selfhosted: make(map[string]*selfhosted.Options),
			},
		},
		"single host for all options should be included": {
			envs: [][2]string{
				{"VERSION_CHECKER_ACR_USERNAME", "acr-username"},
				{"VERSION_CHECKER_ACR_PASSWORD", "acr-password"},
				{"VERSION_CHECKER_ACR_REFRESH_TOKEN", "acr-token"},
				{"VERSION_CHECKER_ACR_JWKS_URI", "acr-jwks-uri"},
				{"VERSION_CHECKER_DOCKER_USERNAME", "docker-username"},
				{"VERSION_CHECKER_DOCKER_PASSWORD", "docker-password"},
				{"VERSION_CHECKER_DOCKER_TOKEN", "docker-token"},
				{"VERSION_CHECKER_ECR_IAM_ROLE_ARN", "iam-role-arn"},
				{"VERSION_CHECKER_ECR_ACCESS_KEY_ID", "ecr-access-token"},
				{"VERSION_CHECKER_ECR_SECRET_ACCESS_KEY", "ecr-secret-access-token"},
				{"VERSION_CHECKER_ECR_SESSION_TOKEN", "ecr-session-token"},
				{"VERSION_CHECKER_GCR_TOKEN", "gcr-token"},
				{"VERSION_CHECKER_GHCR_TOKEN", "ghcr-token"},
				{"VERSION_CHECKER_QUAY_TOKEN", "quay-token"},
				{"VERSION_CHECKER_SELFHOSTED_HOST_FOO", "docker.joshvanl.com"},
				{"VERSION_CHECKER_SELFHOSTED_USERNAME_FOO", "joshvanl"},
				{"VERSION_CHECKER_SELFHOSTED_PASSWORD_FOO", "password"},
				{"VERSION_CHECKER_SELFHOSTED_TOKEN_FOO", "my-token"},
			},
			expOptions: client.Options{
				ACR: acr.Options{
					Username:     "acr-username",
					Password:     "acr-password",
					RefreshToken: "acr-token",
					JWKSURI:      "acr-jwks-uri",
				},
				Docker: docker.Options{
					Username: "docker-username",
					Password: "docker-password",
					Token:    "docker-token",
				},
				ECR: ecr.Options{
					IamRoleArn:      "iam-role-arn",
					AccessKeyID:     "ecr-access-token",
					SecretAccessKey: "ecr-secret-access-token",
					SessionToken:    "ecr-session-token",
				},
				GCR: gcr.Options{
					Token: "gcr-token",
				},
				GHCR: ghcr.Options{
					Token: "ghcr-token",
				},
				Quay: quay.Options{
					Token: "quay-token",
				},
				Selfhosted: map[string]*selfhosted.Options{
					"FOO": {
						Host:     "docker.joshvanl.com",
						Username: "joshvanl",
						Password: "password",
						Bearer:   "my-token",
						Insecure: false,
					},
				},
			},
		},
		"multiple host for all options should be included": {
			envs: [][2]string{
				{"VERSION_CHECKER_SELFHOSTED_HOST_BAR", "bar.docker.joshvanl.com"},
				{"VERSION_CHECKER_SELFHOSTED_USERNAME_BAR", "bar.joshvanl"},
				{"VERSION_CHECKER_SELFHOSTED_PASSWORD_BAR", "bar-password"},
				{"VERSION_CHECKER_SELFHOSTED_TOKEN_BAR", "my-bar-token"},
				{"VERSION_CHECKER_ACR_USERNAME", "acr-username"},
				{"VERSION_CHECKER_ACR_PASSWORD", "acr-password"},
				{"VERSION_CHECKER_ACR_REFRESH_TOKEN", "acr-token"},
				{"VERSION_CHECKER_ACR_JWKS_URI", "acr-jwks-uri"},
				{"VERSION_CHECKER_DOCKER_USERNAME", "docker-username"},
				{"VERSION_CHECKER_DOCKER_PASSWORD", "docker-password"},
				{"VERSION_CHECKER_DOCKER_TOKEN", "docker-token"},
				{"VERSION_CHECKER_ECR_IAM_ROLE_ARN", "iam-role-arn"},
				{"VERSION_CHECKER_ECR_ACCESS_KEY_ID", "ecr-access-token"},
				{"VERSION_CHECKER_ECR_SECRET_ACCESS_KEY", "ecr-secret-access-token"},
				{"VERSION_CHECKER_ECR_SESSION_TOKEN", "ecr-session-token"},
				{"VERSION_CHECKER_GCR_TOKEN", "gcr-token"},
				{"VERSION_CHECKER_GHCR_TOKEN", "ghcr-token"},
				{"VERSION_CHECKER_QUAY_TOKEN", "quay-token"},
				{"VERSION_CHECKER_SELFHOSTED_HOST_FOO", "docker.joshvanl.com"},
				{"VERSION_CHECKER_SELFHOSTED_USERNAME_FOO", "joshvanl"},
				{"VERSION_CHECKER_SELFHOSTED_PASSWORD_FOO", "password"},
				{"VERSION_CHECKER_SELFHOSTED_TOKEN_FOO", "my-token"},
				{"VERSION_CHECKER_SELFHOSTED_INSECURE_FOO", "true"},
				{"VERSION_CHECKER_SELFHOSTED_HOST_BUZZ", "buzz.docker.jetstack.io"},
				{"VERSION_CHECKER_SELFHOSTED_USERNAME_BUZZ", "buzz.davidcollom"},
				{"VERSION_CHECKER_SELFHOSTED_PASSWORD_BUZZ", "buzz-password"},
				{"VERSION_CHECKER_SELFHOSTED_TOKEN_BUZZ", "my-buzz-token"},
				{"VERSION_CHECKER_SELFHOSTED_INSECURE_BUZZ", "false"},
				{"VERSION_CHECKER_SELFHOSTED_CA_PATH_BUZZ", "/var/run/secrets/buzz/ca.crt"},
			},
			expOptions: client.Options{
				ACR: acr.Options{
					Username:     "acr-username",
					Password:     "acr-password",
					RefreshToken: "acr-token",
					JWKSURI:      "acr-jwks-uri",
				},
				Docker: docker.Options{
					Username: "docker-username",
					Password: "docker-password",
					Token:    "docker-token",
				},
				ECR: ecr.Options{
					IamRoleArn:      "iam-role-arn",
					AccessKeyID:     "ecr-access-token",
					SecretAccessKey: "ecr-secret-access-token",
					SessionToken:    "ecr-session-token",
				},
				GCR: gcr.Options{
					Token: "gcr-token",
				},
				GHCR: ghcr.Options{
					Token: "ghcr-token",
				},
				Quay: quay.Options{
					Token: "quay-token",
				},
				Selfhosted: map[string]*selfhosted.Options{
					"FOO": {
						Host:     "docker.joshvanl.com",
						Username: "joshvanl",
						Password: "password",
						Bearer:   "my-token",
						Insecure: true,
					},
					"BAR": {
						Host:     "bar.docker.joshvanl.com",
						Username: "bar.joshvanl",
						Password: "bar-password",
						Bearer:   "my-bar-token",
						Insecure: false,
					},
					"BUZZ": {
						Host:     "buzz.docker.jetstack.io",
						Username: "buzz.davidcollom",
						Password: "buzz-password",
						Bearer:   "my-buzz-token",
						Insecure: false,
						CAPath:   "/var/run/secrets/buzz/ca.crt",
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			os.Clearenv()
			for _, env := range test.envs {
				t.Setenv(env[0], env[1])
			}
			o := new(Options)
			o.complete()

			assert.Exactly(t, test.expOptions, o.Client)
		})
	}
}

func TestInvalidSelfhostedPanic(t *testing.T) {
	tests := map[string]struct {
		envs []string
	}{
		"single host for all options should be included": {
			envs: []string{
				"VERSION_CHECKER_SELFHOSTED_INSECURE_FOO=true",
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			defer func() { _ = recover() }()

			o := new(Options)
			o.assignSelfhosted(test.envs)

			t.Errorf("did not panic")
		})
	}
}

func TestInvalidSelfhostedOpts(t *testing.T) {
	tests := map[string]struct {
		opts  Options
		valid bool
	}{
		"no self hosted configuration": {
			opts:  Options{},
			valid: true,
		},
		"no self hosted host provided": {
			opts: Options{
				Client: client.Options{
					Selfhosted: map[string]*selfhosted.Options{"foo": &selfhosted.Options{
						Insecure: true,
					}},
				},
			},
			valid: false,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			valid := validSelfHostedOpts(&test.opts)

			assert.Equal(t, test.valid, valid)
		})
	}
}

func TestAssignSelfhosted(t *testing.T) {
	tests := map[string]struct {
		envs       []string
		expOptions client.Options
	}{
		"no envs should give no options": {
			envs: []string{},
			expOptions: client.Options{
				Selfhosted: make(map[string]*selfhosted.Options),
			},
		},
		"single host for all options should be included": {
			envs: []string{
				"VERSION_CHECKER_SELFHOSTED_HOST_FOO=docker.joshvanl.com",
				"VERSION_CHECKER_SELFHOSTED_USERNAME_FOO=joshvanl",
				"VERSION_CHECKER_SELFHOSTED_PASSWORD_FOO=password",
				"VERSION_CHECKER_SELFHOSTED_TOKEN_FOO=my-token",
			},
			expOptions: client.Options{
				Selfhosted: map[string]*selfhosted.Options{
					"FOO": {
						Host:     "docker.joshvanl.com",
						Username: "joshvanl",
						Password: "password",
						Bearer:   "my-token",
					},
				},
			},
		},
		"multiple hosts with some values": {
			envs: []string{
				"VERSION_CHECKER_SELFHOSTED_HOST_FOO=docker.joshvanl.com",
				"VERSION_CHECKER_SELFHOSTED_HOST_BAR=hello.world.com",
				"VERSION_CHECKER_SELFHOSTED_USERNAME_FOO=joshvanl",
				"VERSION_CHECKER_SELFHOSTED_PASSWORD_FOO=password",
				"VERSION_CHECKER_SELFHOSTED_TOKEN_FOO=my-token",
				"VERSION_CHECKER_SELFHOSTED_TOKEN_BAR=my-bar-token",
			},
			expOptions: client.Options{
				Selfhosted: map[string]*selfhosted.Options{
					"FOO": {
						Host:     "docker.joshvanl.com",
						Username: "joshvanl",
						Password: "password",
						Bearer:   "my-token",
					},
					"BAR": {
						Host:   "hello.world.com",
						Bearer: "my-bar-token",
					},
				},
			},
		},
		"allow token path override": {
			envs: []string{
				"VERSION_CHECKER_SELFHOSTED_HOST_FOO=docker.joshvanl.com",
				"VERSION_CHECKER_SELFHOSTED_HOST_BAR=hello.world.com",
				"VERSION_CHECKER_SELFHOSTED_USERNAME_FOO=joshvanl",
				"VERSION_CHECKER_SELFHOSTED_PASSWORD_FOO=password",
				"VERSION_CHECKER_SELFHOSTED_TOKEN_FOO=my-token",
				"VERSION_CHECKER_SELFHOSTED_TOKEN_BAR=my-bar-token",
				"VERSION_CHECKER_SELFHOSTED_TOKEN_PATH_FOO=/artifactory/api/security/token",
			},
			expOptions: client.Options{
				Selfhosted: map[string]*selfhosted.Options{
					"FOO": {
						Host:      "docker.joshvanl.com",
						Username:  "joshvanl",
						Password:  "password",
						Bearer:    "my-token",
						TokenPath: "/artifactory/api/security/token",
					},
					"BAR": {
						Host:   "hello.world.com",
						Bearer: "my-bar-token",
					},
				},
			},
		},
		"ignore keys with no values": {
			envs: []string{
				"VERSION_CHECKER_SELFHOSTED_HOST_FOO=docker.joshvanl.com",
				"VERSION_CHECKER_SELFHOSTED_HOST_BAR=hello.world.com",
				"VERSION_CHECKER_SELFHOSTED_USERNAME_FOO=joshvanl",
				"VERSION_CHECKER_SELFHOSTED_PASSWORD_FOO=password",
				"VERSION_CHECKER_SELFHOSTED_TOKEN_FOO=my-token",
				"VERSION_CHECKER_SELFHOSTED_TOKEN_BAR=my-bar-token",
				"VERSION_CHECKER_SELFHOSTED_TOKEN_HELLO=",
				"VERSION_CHECKER_SELFHOSTED_HOST_HELLO",
				"VERSION_CHECKER_SELFHOSTED_HOST_joshvanl=joshvanl.com",
			},
			expOptions: client.Options{
				Selfhosted: map[string]*selfhosted.Options{
					"FOO": {
						Host:     "docker.joshvanl.com",
						Username: "joshvanl",
						Password: "password",
						Bearer:   "my-token",
					},
					"BAR": {
						Host:   "hello.world.com",
						Bearer: "my-bar-token",
					},
					"JOSHVANL": {
						Host: "joshvanl.com",
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			o := new(Options)
			o.assignSelfhosted(test.envs)

			assert.Exactly(t, test.expOptions.Selfhosted, o.Client.Selfhosted)
		})
	}
}
