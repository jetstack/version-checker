package util

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jetstack/version-checker/pkg/api"
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

func TestOSArchFromTag(t *testing.T) {
	tests := map[string]struct {
		tag     string
		expOS   api.OS
		expArch api.Architecture
	}{
		"empty tag should return empty OS and Arch": {
			tag:     "",
			expOS:   "",
			expArch: "",
		},
		"tag with only OS should return correct OS and empty Arch": {
			tag:     "v1.0.0-linux",
			expOS:   "linux",
			expArch: "",
		},
		"tag with only Arch should return linux OS and correct Arch": {
			tag:     "v1.0.0-amd64",
			expOS:   "linux",
			expArch: "amd64",
		},
		"tag with OS and Arch should return both correctly": {
			tag:     "v1.0.0-linux-amd64",
			expOS:   "linux",
			expArch: "amd64",
		},
		"tag with unknown OS and Arch should return empty OS and Arch": {
			tag:     "v1.0.0-os-unknown-arch",
			expOS:   "",
			expArch: "",
		},
		"tag with mixed case OS and Arch should return correct OS and Arch": {
			tag:     "v1.0.0-Linux-AMD64",
			expOS:   "linux",
			expArch: "amd64",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			os, arch := OSArchFromTag(test.tag)

			assert.Equal(t, os, test.expOS)
			assert.Equal(t, arch, test.expArch)
		})
	}
}
func TestTagMaptoList(t *testing.T) {
	tests := map[string]struct {
		tags    map[string]api.ImageTag
		expList []api.ImageTag
	}{
		"empty map should return empty list": {
			tags:    map[string]api.ImageTag{},
			expList: []api.ImageTag{},
		},
		"single entry map should return single element list": {
			tags: map[string]api.ImageTag{
				"v1.0.0": {Tag: "v1.0.0"},
			},
			expList: []api.ImageTag{
				{Tag: "v1.0.0"},
			},
		},
		"multiple entry map should return list with all elements": {
			tags: map[string]api.ImageTag{
				"v1.0.0": {Tag: "v1.0.0"},
				"v1.1.0": {Tag: "v1.1.0"},
				"v2.0.0": {Tag: "v2.0.0"},
			},
			expList: []api.ImageTag{
				{Tag: "v1.0.0"},
				{Tag: "v1.1.0"},
				{Tag: "v2.0.0"},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := TagMaptoList(test.tags)
			assert.ElementsMatch(t, result, test.expList)
		})
	}
}
