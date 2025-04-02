package ecr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsHost(t *testing.T) {
	tests := map[string]struct {
		host  string
		expIs bool
	}{
		"an empty host should be false": {
			host:  "",
			expIs: false,
		},
		"random string should be false": {
			host:  "foobar",
			expIs: false,
		},
		"random string with dots should be false": {
			host:  "foobar.foo",
			expIs: false,
		},
		"just amazonawsaws.com should be false": {
			host:  "amazonaws.com",
			expIs: false,
		},
		"ecr.foo.amazonaws.com with random sub domains should be false": {
			host:  "bar.ecr.foo.amazonaws.com",
			expIs: false,
		},
		"dkr.ecr.foo.amazonaws.com with random sub domains should be false": {
			host:  "dkr.ecr.foo.amazonaws.com",
			expIs: false,
		},
		"hello123.dkr.ecr.foo.amazonaws.com true": {
			host:  "hello123.dkr.ecr.foo.amazonaws.com",
			expIs: true,
		},
		"123hello.dkr.ecr.foo.amazonaws.com true": {
			host:  "123hello.dkr.ecr.foo.amazonaws.com",
			expIs: true,
		},
		"hello123.dkr.ecr.foo.amazonaws.com.cn true": {
			host:  "hello123.dkr.ecr.foo.amazonaws.com.cn",
			expIs: true,
		},
		"123hello.dkr.ecr.foo.amazonaws.com.cn true": {
			host:  "123hello.dkr.ecr.foo.amazonaws.com.cn",
			expIs: true,
		},
		"123hello.hello.dkr.ecr.foo.amazonaws.com false": {
			host:  "123hello.hello.dkr.ecr.foo.amazonaws.com",
			expIs: false,
		},
		"123hello.dkr.ecr.foo.amazonaws.comfoo false": {
			host:  "123hello.dkr.ecr.foo.amazonaws.comfoo",
			expIs: false,
		},
	}

	handler := new(Client)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if isHost := handler.IsHost(test.host); isHost != test.expIs {
				t.Errorf("%s: unexpected IsHost, exp=%t got=%t",
					test.host, test.expIs, isHost)
			}
		})
	}
}

func TestRepoImage(t *testing.T) {
	tests := map[string]struct {
		path              string
		expRepo, expImage string
	}{
		"single image should return as image": {
			path:     "kube-scheduler",
			expRepo:  "",
			expImage: "kube-scheduler",
		},
		"two segments to path should return both": {
			path:     "jetstack-cre/version-checker",
			expRepo:  "jetstack-cre",
			expImage: "version-checker",
		},
		"multiple segments to path should return all in repo, last segment image": {
			path:     "k8s-artifacts-prod/ingress-nginx/nginx",
			expRepo:  "k8s-artifacts-prod/ingress-nginx",
			expImage: "nginx",
		},
		"region": {
			path:     "000000000000.dkr.ecr.eu-west-2.amazonaws.com/version-checker",
			expRepo:  "000000000000.dkr.ecr.eu-west-2.amazonaws.com",
			expImage: "version-checker",
		},
	}

	handler := new(Client)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			repo, image := handler.RepoImageFromPath(test.path)
			assert.Equal(t, repo, test.expRepo)
			assert.Equal(t, image, test.expImage)
		})
	}
}
