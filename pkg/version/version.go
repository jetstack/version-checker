package version

import (
	"context"
	"errors"
	"fmt"

	"github.com/masterminds/semver"

	"github.com/joshvanl/version-checker/pkg/api"
	"github.com/joshvanl/version-checker/pkg/version/docker"
	"github.com/joshvanl/version-checker/pkg/version/gcr"
	"github.com/joshvanl/version-checker/pkg/version/quay"
)

var (
	zeroVersion = semver.MustParse("0.0.0")
)

type VersionGetter struct {
	quay   *quay.Client
	docker *docker.Client
	gcr    *gcr.Client
}

// TODO: add comments to funcs
func New() *VersionGetter {
	return &VersionGetter{
		quay:   quay.New(),
		docker: docker.New(),
		gcr:    gcr.New(),
	}
}

// LatestTagFromImage will return the latest image tag from the remote registry
// given an image URL, along with the full set of tags found for that image.
func (v *VersionGetter) LatestTagFromImage(ctx context.Context, options *api.Options, imageURL string) (*api.ImageTag, []api.ImageTag, error) {
	client := v.clientFromImage(imageURL)

	tags, err := client.Tags(ctx, imageURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get tags from remote registry for %q: %s",
			imageURL, err)
	}

	if len(tags) == 0 {
		return nil, nil, fmt.Errorf("no tags found for given image URL: %q", imageURL)
	}

	latestTag, err := latestTag(options, tags)
	if err != nil {
		return nil, nil, err
	}

	return latestTag, tags, nil
}

// clientFromImage will return the appropriate registry client for a given
// image URL.
func (v *VersionGetter) clientFromImage(imageURL string) api.ImageClient {
	switch {
	case v.quay.IsClient(imageURL):
		return v.quay
	case v.gcr.IsClient(imageURL):
		return v.gcr
	case v.docker.IsClient(imageURL):
		return v.docker
	default:
		// Fall back to docker if we can't determine the registry
		return v.docker
	}
}

// latestTag will return the latest tag given a set of tags, according to the
// given options.
func latestTag(options *api.Options, tags []api.ImageTag) (*api.ImageTag, error) {
	// If UseSHA then return early
	if options.UseSHA {
		return latestSHA(tags)
	}

	var latest api.ImageTag

	for i, tag := range tags {
		v, err := semver.NewVersion(tag.Tag)
		if err == semver.ErrInvalidSemVer {
			continue
		}
		if err != nil {
			return nil, err
		}

		tags[i].SemVer = v

		// If regex enabled but doesn't match tag, continue
		if options.RegexMatcher != nil && !options.RegexMatcher.MatchString(tag.Tag) {
			continue
		}

		// Optionally use pre-release
		if v.Prerelease() != "" && !options.UsePreRelease {
			continue
		}

		if options.PinMajor != nil && v.Major() != *options.PinMajor {
			continue
		}
		if options.PinMinor != nil && v.Minor() != *options.PinMinor {
			continue
		}
		if options.PinPatch != nil && v.Patch() != *options.PinPatch {
			continue
		}

		if latest.SemVer == nil || latest.SemVer.LessThan(v) {
			latest = tags[i]
		}
	}

	if latest.SemVer == nil {
		return nil, fmt.Errorf("no image found with those option contraints: %+v", options)
	}

	return &latest, nil
}

func latestSHA(tags []api.ImageTag) (*api.ImageTag, error) {
	var tag *api.ImageTag

	for i := range tags {
		if tag == nil || tags[i].Timestamp.After(tag.Timestamp) {
			tag = &tags[i]
		}
	}

	if tag == nil {
		return nil, errors.New("failed to find latest image based on SHA")
	}

	return tag, nil
}
