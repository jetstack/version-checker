package app

import (
	"os"
	"reflect"
	"testing"

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
					"FOO": &selfhosted.Options{
						Host:     "docker.joshvanl.com",
						Username: "joshvanl",
						Password: "password",
						Bearer:   "my-token",
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
					"FOO": &selfhosted.Options{
						Host:     "docker.joshvanl.com",
						Username: "joshvanl",
						Password: "password",
						Bearer:   "my-token",
					},
					"BAR": &selfhosted.Options{
						Host:     "bar.docker.joshvanl.com",
						Username: "bar.joshvanl",
						Password: "bar-password",
						Bearer:   "my-bar-token",
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, env := range test.envs {
				os.Setenv(env[0], env[1])
			}
			o := new(Options)
			o.complete()

			if !reflect.DeepEqual(o.Client, test.expOptions) {
				t.Errorf("unexpected client options, exp=%#+v got=%#+v",
					test.expOptions, o.Client)
			}

			for _, env := range test.envs {
				os.Unsetenv(env[0])
			}
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
					"FOO": &selfhosted.Options{
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
					"FOO": &selfhosted.Options{
						Host:     "docker.joshvanl.com",
						Username: "joshvanl",
						Password: "password",
						Bearer:   "my-token",
					},
					"BAR": &selfhosted.Options{
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
					"FOO": &selfhosted.Options{
						Host:     "docker.joshvanl.com",
						Username: "joshvanl",
						Password: "password",
						Bearer:   "my-token",
					},
					"BAR": &selfhosted.Options{
						Host:   "hello.world.com",
						Bearer: "my-bar-token",
					},
					"JOSHVANL": &selfhosted.Options{
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

			if !reflect.DeepEqual(o.Client.Selfhosted, test.expOptions.Selfhosted) {
				t.Errorf("unexpected client selfhosted options, exp=%#+v got=%#+v",
					test.expOptions.Selfhosted, o.Client.Selfhosted)
			}
		})
	}
}
