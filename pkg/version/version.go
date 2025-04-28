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

var _ cache.Handler = (*Version)(nil)

type Version struct {
	log *logrus.Entry

	client     client.ClientHandler
	imageCache *cache.Cache
}

func New(log *logrus.Entry, client client.ClientHandler, cacheTimeout time.Duration) *Version {
	log = log.WithField("module", "version_getter")

	v := &Version{
		log:    log,
		client: client,
	}

	v.imageCache = cache.New(log, cacheTimeout, v)

	return v
}

// LatestTagFromImage will return the latest tag given an imageURL, according
// to the given options.
func (v *Version) LatestTagFromImage(ctx context.Context, imageURL string, opts *api.Options) (*api.ImageTag, error) {
	tagsI, err := v.imageCache.Get(ctx, imageURL, imageURL, nil)
	if err != nil {
		return nil, err
	}
	tags := tagsI.([]api.ImageTag)

	var tag *api.ImageTag

	// If UseSHA then return early
	if opts.UseSHA {
		tag, err = latestSHA(opts, tags)
		if err != nil {
			return nil, err
		}

		if tag == nil {
			return nil, versionerrors.NewVersionErrorNotFound(
				"%s: failed to find latest image based on SHA", imageURL)
		}
	} else {
		tag, err = latestSemver(opts, tags)
		if err != nil {
			return nil, err
		}

		if tag == nil {
			optsBytes, _ := json.Marshal(opts)
			return nil, versionerrors.NewVersionErrorNotFound(
				"%s: no tags found with these option constraints: %s",
				imageURL, optsBytes)
		}
	}

	return tag, err
}

// ResolveSHAToTag Resolve a SHA to a tag if possible
func (v *Version) ResolveSHAToTag(ctx context.Context, imageURL string, imageSHA string) (string, error) {
	tagsI, err := v.imageCache.Get(ctx, imageURL, imageURL, nil)
	if err != nil {
		return "", err
	}
	tags := tagsI.([]api.ImageTag)

	for i := range tags {
		if tags[i].MatchesSHA(imageSHA) {
			return tags[i].Tag, nil
		}
	}

	return "", nil
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
	v.log.WithField("image", imageURL).Debugf("fetched %v tags", len(tags))

	return tags, nil
}
