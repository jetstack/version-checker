package version

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/joshvanl/version-checker/pkg/api"
	"github.com/joshvanl/version-checker/pkg/client"
	"github.com/joshvanl/version-checker/pkg/version/semver"
)

type VersionGetter struct {
	log *logrus.Entry

	client *client.Client

	// cacheTimeout is the amount of time a imageCache item is considered fresh
	// for.
	cacheTimeout time.Duration
	cacheMu      sync.RWMutex
	imageCache   map[string]imageCacheItem
}

func New(log *logrus.Entry, client *client.Client, cacheTimeout time.Duration) *VersionGetter {
	vg := &VersionGetter{
		log:          log.WithField("module", "version_getter"),
		client:       client,
		imageCache:   make(map[string]imageCacheItem),
		cacheTimeout: cacheTimeout,
	}

	// Start garbage collector
	go vg.garbageCollect(cacheTimeout / 2)

	return vg
}

// LatestTagFromOImage will return the latest tag given an imageURL, according
// to the given options.
func (v *VersionGetter) LatestTagFromImage(ctx context.Context, opts *api.Options, imageURL string) (*api.ImageTag, error) {
	tags, err := v.allTagsFromImage(ctx, imageURL)
	if err != nil {
		return nil, err
	}

	// If UseSHA then return early
	if opts.UseSHA {
		return latestSHA(tags)
	}

	return latestSemver(opts, tags)
}

// allTagsFromImage will return all available tags from the remote repository
// given an imageURL. It also holds a cache for each imageURL that is
// periodically garbage collected.
func (v *VersionGetter) allTagsFromImage(ctx context.Context, imageURL string) ([]api.ImageTag, error) {
	// Check for cache hit
	if tags, ok := v.tryImageCache(imageURL); ok {
		return tags, nil
	}

	// Cache miss so pull fresh tags
	tags, err := v.client.Tags(ctx, imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags from remote registry for %q: %s",
			imageURL, err)
	}

	if len(tags) == 0 {
		return nil, fmt.Errorf("no tags found for given image URL: %q", imageURL)
	}

	v.log.Debugf("committing image tags: %q", imageURL)

	// Add tags to cache
	v.imageCache[imageURL] = imageCacheItem{
		timestamp: time.Now(),
		tags:      tags,
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

		// If regex enabled continue here.
		// If we match, and is less than, update latest.
		if opts.RegexMatcher != nil {
			if opts.RegexMatcher.MatchString(tags[i].Tag) &&
				(latestV == nil || latestV.LessThan(v)) {
				latestV = v
				latestImageTag = &tags[i]
			}

			continue
		}

		// If we have declared we wont use metadata but version has it, continue.
		if !opts.UseMetaData && v.HasMetaData() {
			continue
		}

		if opts.PinMajor != nil && *opts.PinMajor != v.Major() {
			continue
		}
		if opts.PinMinor != nil && *opts.PinMinor != v.Minor() {
			continue
		}
		if opts.PinPatch != nil && *opts.PinPatch != v.Patch() {
			continue
		}

		if latestV == nil || latestV.LessThan(v) {
			latestV = v
			latestImageTag = &tags[i]
		}
	}

	if latestImageTag == nil {
		return nil, fmt.Errorf("no tag found with those option constraints: %+v", opts)
	}

	return latestImageTag, nil
}

// latestSHA will return the latest ImageTag based on image timestamps.
func latestSHA(tags []api.ImageTag) (*api.ImageTag, error) {
	var latestTag *api.ImageTag

	for i := range tags {
		if latestTag == nil || tags[i].Timestamp.After(latestTag.Timestamp) {
			latestTag = &tags[i]
		}
	}

	if latestTag == nil {
		return nil, errors.New("failed to find latest image based on SHA")
	}

	return latestTag, nil
}
