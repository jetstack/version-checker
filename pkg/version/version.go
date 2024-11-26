package version

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client"

	"github.com/jetstack/version-checker/pkg/cache"
	versionerrors "github.com/jetstack/version-checker/pkg/version/errors"
	"github.com/jetstack/version-checker/pkg/version/semver"
)

type Version struct {
	log *logrus.Entry

	client     *client.Client
	imageCache *cache.Cache
}

func New(log *logrus.Entry, client *client.Client, cacheTimeout time.Duration) *Version {
	log = log.WithField("module", "version_getter")

	v := &Version{
		log:    log,
		client: client,
	}

	v.imageCache = cache.New(log, cacheTimeout, v)

	return v
}

// Run is a blocking func that will start the image cache garbage collector.
func (v *Version) Run(refreshRate time.Duration) {
	v.imageCache.StartGarbageCollector(refreshRate)
}

// AllTagsFromImage will return all tags given an imageURL.
func (v *Version) AllTagsFromImage(ctx context.Context, imageURL string) ([]api.ImageTag, error) {
	if tagsI, err := v.imageCache.Get(ctx, imageURL, imageURL, nil); err != nil {
		return nil, err
	} else {
		return tagsI.([]api.ImageTag), err
	}
}

// LatestTagFromImage will return the latest tag given an imageURL, according
// to the given options.
func (v *Version) LatestTagFromImage(ctx context.Context, imageURL string, opts *api.Options) (*api.ImageTag, error) {
	tags, err := v.AllTagsFromImage(ctx, imageURL)
	if err != nil {
		return nil, err
	}

	var tag *api.ImageTag

	// If UseSHA then return early
	if opts.UseSHA {
		tag, err = latestSHA(tags)
		if err != nil {
			return nil, err
		}

		if tag == nil {
			return nil, versionerrors.NewVersionErrorNotFound("%s: failed to find latest image based on SHA",
				imageURL)
		}
	} else {
		tag, err = latestSemver(opts, tags)
		if err != nil {
			return nil, err
		}

		if tag == nil {
			optsBytes, _ := json.Marshal(opts)
			return nil, versionerrors.NewVersionErrorNotFound("%s: no tags found with these option constraints: %s",
				imageURL, optsBytes)
		}
	}

	return tag, err
}

// Fetch returns the given image tags for a given image URL.
func (v *Version) Fetch(ctx context.Context, imageURL string, _ *api.Options) (interface{}, error) {
	// fetch tags from image URL
	tags, err := v.client.Tags(ctx, imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags from remote registry for %q: %s",
			imageURL, err)
	}

	// respond with no version found if no manifests were found to prevent
	// needlessly querying a bad URL.
	if len(tags) == 0 {
		return nil, versionerrors.NewVersionErrorNotFound("no tags found for given image URL: %q", imageURL)
	}

	return tags, nil
}

// latestSemver will return the latest ImageTag based on the given options
// restriction, using semver. This should not be used is UseSHA has been
// enabled.
func latestSemver(opts *api.Options, tags []api.ImageTag) (*api.ImageTag, error) {
	var (
		latestImageTag *api.ImageTag
		latestV        *semver.SemVer
	)

	for i := range tags {
		v := semver.Parse(tags[i].Tag)

		if shouldSkipTag(opts, v) {
			continue
		}

		if isBetterTag(opts, latestV, v, latestImageTag, &tags[i]) {
			latestV = v
			latestImageTag = &tags[i]
		}
	}

	if latestImageTag == nil {
		return nil, fmt.Errorf("no suitable version found")
	}

	return latestImageTag, nil
}

func shouldSkipTag(opts *api.Options, v *semver.SemVer) bool {
	// Handle Regex matching
	if opts.RegexMatcher != nil {
		return !opts.RegexMatcher.MatchString(v.String())
	}

	// Handle metadata and version pinning
	return (!opts.UseMetaData && v.HasMetaData()) ||
		(opts.PinMajor != nil && *opts.PinMajor != v.Major()) ||
		(opts.PinMinor != nil && *opts.PinMinor != v.Minor()) ||
		(opts.PinPatch != nil && *opts.PinPatch != v.Patch())
}

func isBetterTag(_ *api.Options, latestV, v *semver.SemVer, latestImageTag, currentImageTag *api.ImageTag) bool {
	// No latest version set yet
	if latestV == nil {
		return true
	}

	// If the current version is greater than the latest
	if latestV.LessThan(v) {
		return true
	}

	// If the versions are equal, prefer the one with a later timestamp
	if latestV.Equal(v) && currentImageTag.Timestamp.After(latestImageTag.Timestamp) {
		return true
	}

	return false
}

// latestSHA will return the latest ImageTag based on image timestamps.
func latestSHA(tags []api.ImageTag) (*api.ImageTag, error) {
	var latestTag *api.ImageTag

	for i := range tags {
		if latestTag == nil || tags[i].Timestamp.After(latestTag.Timestamp) {
			latestTag = &tags[i]
		}
	}

	return latestTag, nil
}
