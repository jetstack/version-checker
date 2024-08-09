package util

import (
	"strings"

	"github.com/jetstack/version-checker/pkg/api"
)

var (
	KnownOSs = [...]api.OS{
		"linux",
		"darwin",
		"windows",
		"freebsd",
	}

	KnownArchs = [...]api.Architecture{
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

// Join repo and image strings
func JoinRepoImage(repo, image string) string {
	if len(repo) == 0 {
		return image
	}
	if len(image) == 0 {
		return repo
	}

	return repo + "/" + image
}

// Attempt to determine the OS and Arch, given a tag name
func OSArchFromTag(tag string) (api.OS, api.Architecture) {
	var (
		os    api.OS
		arch  api.Architecture
		split = strings.Split(tag, "-")
	)

	for _, s := range split {
		ss := strings.ToLower(s)

		for _, pos := range KnownOSs {
			if pos == api.OS(ss) {
				os = pos
			}
		}

		for _, parch := range KnownArchs {
			if parch == api.Architecture(ss) {
				arch = parch
			}
		}
	}

	return os, arch
}

func FilterSbomAttestationSigs(tag string) bool {
	return strings.HasSuffix(tag, ".att") || strings.HasSuffix(tag, ".sig") || strings.HasSuffix(tag, ".sbom")
}
