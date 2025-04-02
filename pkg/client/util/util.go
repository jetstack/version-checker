package util

import (
	"strings"

	"github.com/jetstack/version-checker/pkg/api"
)

var (
	oss = [...]api.OS{
		"linux",
		"darwin",
		"windows",
		"freebsd",
	}

	archs = [...]api.Architecture{
		"amd",
		"amd64",
		"arm",
		"arm64",
		"arm32v5",
		"arm32v6",
		"arm32v7",
		"arm64v8",
		"i386",
		"ppc64",
		"ppc64le",
		"s390x",
		"x86",
		"x86_64",
		"mips",
	}
)

// Join repo and image strings.
func JoinRepoImage(repo, image string) string {
	if len(repo) == 0 {
		return image
	}
	if len(image) == 0 {
		return repo
	}

	return repo + "/" + image
}

// Attempt to determine the OS and Arch, given a tag name.
func OSArchFromTag(tag string) (api.OS, api.Architecture) {
	var (
		os    api.OS
		arch  api.Architecture
		split = strings.Split(tag, "-")
	)

	for _, s := range split {
		ss := strings.ToLower(s)

		for _, pos := range oss {
			if pos == api.OS(ss) {
				os = pos
			}
		}

		for _, parch := range archs {
			if parch == api.Architecture(ss) {
				arch = parch
			}
		}
	}

	return os, arch
}
