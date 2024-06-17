package version

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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

// LatestTagFromImage will return the latest tag given an imageURL, according
// to the given options.
func (v *Version) LatestTagFromImage(ctx context.Context, imageURL string, opts *api.Options) (*api.ImageTag, error) {
	if override := opts.OverrideURL; override != nil && len(*override) > 0 {
		v.log.Debugf("overriding image lookup %s -> %s", imageURL, *override)
		imageURL = *override
	}
	tagsI, err := v.imageCache.Get(ctx, imageURL, imageURL, nil)
	if err != nil {
		return nil, err
	}
	tags := tagsI.([]api.ImageTag)

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
// Function to find the latest SemVer tag based on options
func latestSemver(opts *api.Options, tags []api.ImageTag) (*api.ImageTag, error) {
	var filteredTags []api.ImageTag

	// Filter out non-SemVer tags if required
	if !opts.UseMetaData {
		for _, tag := range tags {
			if isSemVer(tag.Tag) {
				filteredTags = append(filteredTags, tag)
			}
		}
	} else {
		filteredTags = tags
	}

	// Apply regex matching if provided
	if opts.RegexMatcher != nil {
		var matchedTags []api.ImageTag
		for _, tag := range filteredTags {
			if opts.RegexMatcher.MatchString(tag.Tag) {
				matchedTags = append(matchedTags, tag)
			}
		}
		filteredTags = matchedTags
	}

	// Convert tags to semver.Version instances for sorting
	var versions []*semver.Version
	for _, tag := range filteredTags {
		v, err := semver.NewVersion(tag.Tag)
		if err == nil {
			versions = append(versions, v)
		}
	}

	// If no valid SemVer tags are found, return an error
	if len(versions) == 0 {
		return nil, fmt.Errorf("no matching SemVer tags found")
	}

	// Sort versions by descending order
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].GreaterThan(versions[j])
	})

	// Apply version pinning if provided
	if opts.PinMajor != nil || opts.PinMinor != nil || opts.PinPatch != nil {
		for _, v := range versions {
			if versionMatches(v, opts.PinMajor, opts.PinMinor, opts.PinPatch) {
				return &api.ImageTag{Tag: v.Original()}, nil
			}
		}
	}

	// Return the latest SemVer tag
	return &api.ImageTag{Tag: versions[0].Original()}, nil
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

// Helper function to check if a version matches the pinned version
func versionMatches(v *semver.Version, pinMajor, pinMinor, pinPatch *int64) bool {
	if pinMajor != nil && *pinMajor != int64(v.Major()) {
		return false
	}
	if pinMinor != nil && *pinMinor != int64(v.Minor()) {
		return false
	}
	if pinPatch != nil && *pinPatch != int64(v.Patch()) {
		return false
	}
	return true
}
