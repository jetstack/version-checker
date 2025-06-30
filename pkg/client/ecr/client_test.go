package ecr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
