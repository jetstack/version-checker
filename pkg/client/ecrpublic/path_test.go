package ecrpublic

import "testing"

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
		"path with two segments should be false": {
			host:  "joshvanl/version-checker",
			expIs: false,
		},
		"path with three segments should be false": {
			host:  "jetstack/joshvanl/version-checker",
			expIs: false,
		},
		"random string with dots should be false": {
			host:  "foobar.foo",
			expIs: false,
		},
		"docker.io should be false": {
			host:  "docker.io",
			expIs: false,
		},
		"docker.com should be false": {
			host:  "docker.com",
			expIs: false,
		},
		"just public.ecr.aws should be true": {
			host:  "public.ecr.aws",
			expIs: true,
		},
		"public.ecr.aws.foo should be false": {
			host:  "public.ecr.aws.foo",
			expIs: false,
		},
		"foo.public.ecr.aws should be false": {
			host:  "foo.public.ecr.aws",
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

func TestRepoImageFromPath(t *testing.T) {
	tests := map[string]struct {
		path              string
		expRepo, expImage string
	}{
		"single image should return registry and image": {
			path:     "nginx",
			expRepo:  "nginx",
			expImage: "",
		},
		"two segments to path should return registry and repo": {
			path:     "eks-distro/kubernetes",
			expRepo:  "eks-distro",
			expImage: "kubernetes",
		},
		"three segments to path should return registry and combined repo": {
			path:     "eks-distro/kubernetes/kube-proxy",
			expRepo:  "eks-distro",
			expImage: "kubernetes/kube-proxy",
		},
	}

	handler := new(Client)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			repo, image := handler.RepoImageFromPath(test.path)
			if repo != test.expRepo || image != test.expImage {
				t.Errorf("%s: unexpected repo/image, exp=%s/%s got=%s/%s",
					test.path, test.expRepo, test.expImage, repo, image)
			}
		})
	}
}
