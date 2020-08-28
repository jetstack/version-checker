package util

import (
	"testing"
)

func TestJoinRepoImage(t *testing.T) {
	tests := map[string]struct {
		repo, image string
		expPath     string
	}{
		"single image should return as image": {
			repo:    "",
			image:   "kube-scheduler",
			expPath: "kube-scheduler",
		},
		"single repo should return as repo": {
			repo:    "kube-scheduler",
			image:   "",
			expPath: "kube-scheduler",
		},
		"two segments to path should return both": {
			repo:    "jetstack-cre",
			image:   "version-checker",
			expPath: "jetstack-cre/version-checker",
		},
		"multiple segments to repo should return all in repo and image": {
			repo:    "k8s-artifacts-prod/ingress-nginx",
			image:   "nginx",
			expPath: "k8s-artifacts-prod/ingress-nginx/nginx",
		},
		"multiple segments to image should return all in repo and image": {
			repo:    "k8s-artifacts-prod",
			image:   "ingress-nginx/nginx",
			expPath: "k8s-artifacts-prod/ingress-nginx/nginx",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if path := JoinRepoImage(test.repo, test.image); path != test.expPath {
				t.Errorf("%s,%s: unexpected path, exp=%s got=%s",
					test.repo, test.repo, test.expPath, path)
			}
		})
	}
}
