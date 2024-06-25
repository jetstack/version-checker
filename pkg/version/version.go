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
		tag, err = latestSHA(tags, opts)
		if err != nil {
			return nil, err
		}

		if tag == nil {
			return nil, versionerrors.NewVersionErrorNotFound("%s: failed to find latest image based on SHA",
				imageURL)
		}

	} else {
		tag, err = latestSemver(tags, opts)
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
