package checker

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/controller/search"
	"github.com/jetstack/version-checker/pkg/version/semver"
	"github.com/sirupsen/logrus"
)

type Checker struct {
	search search.Searcher
}

type Result struct {
	CurrentVersion string
	LatestVersion  string
	IsLatest       bool
	ImageURL       string
}

func New(search search.Searcher) *Checker {
	return &Checker{
		search: search,
	}
}

// Container will return the result of the given container's current version, compared to the latest upstream.
func (c *Checker) Container(ctx context.Context, log *logrus.Entry,
	pod *corev1.Pod,
	container *corev1.Container,
	opts *api.Options,
) (*Result, error) {
	statusSHA := containerStatusImageSHA(pod, container.Name)
	if len(statusSHA) == 0 {
		return nil, nil
	}

	imageURL, currentTag, currentSHA := urlTagSHAFromImage(container.Image)
	usingSHA, usingTag := len(currentSHA) > 0, len(currentTag) > 0

	if opts.ResolveSHAToTags {

		if len(*opts.OverrideURL) > 0 {
			imageURL = *opts.OverrideURL
		}
		resolvedTag, err := c.search.ResolveSHAToTag(ctx, imageURL, currentSHA)

		if len(resolvedTag) > 0 && err == nil {
			log.Infof("Successfully resolved tag for sha256: %s at url: %s", currentSHA, imageURL)
			currentTag = resolvedTag
			usingSHA = false
			usingTag = true
		}
	}

	// If using latest or no tag, then compare on SHA
	if c.isLatestOrEmptyTag(currentTag) {
		c.handleLatestOrEmptyTag(log, currentTag, currentSHA, opts)
		usingTag = false
	}

	imageURL = c.overrideImageURL(log, imageURL, opts)

	if opts.UseSHA {
		return c.handleSHA(ctx, imageURL, statusSHA, opts, usingTag, currentTag)
	}

	return c.handleSemver(ctx, imageURL, statusSHA, currentTag, usingSHA, opts)
}

func (c *Checker) handleLatestOrEmptyTag(log *logrus.Entry, currentTag, currentSHA string, opts *api.Options) {
	opts.UseSHA = true
	log.WithField("module", "checker").Debugf("image using %q tag, comparing image SHA %q", currentTag, currentSHA)
}

func (c *Checker) overrideImageURL(log *logrus.Entry, imageURL string, opts *api.Options) string {
	if opts.OverrideURL != nil && *opts.OverrideURL != imageURL {
		log.Debugf("overriding image URL %s -> %s", imageURL, *opts.OverrideURL)
		return *opts.OverrideURL
	}
	return imageURL
}

func (c *Checker) handleSHA(ctx context.Context, imageURL, statusSHA string, opts *api.Options, usingTag bool, currentTag string) (*Result, error) {
	result, err := c.isLatestSHA(ctx, imageURL, statusSHA, opts)
	if err != nil {
		return nil, err
	}

	if usingTag {
		result.CurrentVersion = fmt.Sprintf("%s@%s", currentTag, result.CurrentVersion)
	}

	return result, nil
}

func (c *Checker) handleSemver(ctx context.Context, imageURL, statusSHA, currentTag string, usingSHA bool, opts *api.Options) (*Result, error) {
	currentImage := semver.Parse(currentTag)
	latestImage, isLatest, err := c.isLatestSemver(ctx, imageURL, statusSHA, currentImage, opts)
	if err != nil {
		return nil, err
	}

	latestVersion := latestImage.Tag
	if usingSHA && !strings.Contains(latestVersion, "@") && latestImage.SHA != "" {
		latestVersion = fmt.Sprintf("%s@%s", latestVersion, latestImage.SHA)
	}

	if strings.Contains(latestVersion, "@") {
		currentTag = fmt.Sprintf("%s@%s", currentTag, statusSHA)
	}

	return &Result{
		CurrentVersion: currentTag,
		LatestVersion:  latestVersion,
		IsLatest:       isLatest,
		ImageURL:       imageURL,
	}, nil
}

// isLatestOrEmptyTag will return true if the given tag is "" or "latest".
func (c *Checker) isLatestOrEmptyTag(tag string) bool {
	return tag == "" || tag == "latest"
}

// isLatestSemver will return the latest image, and whether the given image is the latest.
func (c *Checker) isLatestSemver(ctx context.Context, imageURL, currentSHA string, currentImage *semver.SemVer, opts *api.Options) (*api.ImageTag, bool, error) {
	latestImage, err := c.search.LatestImage(ctx, imageURL, opts)
	if err != nil {
		return nil, false, err
	}

	latestImageV := semver.Parse(latestImage.Tag)

	var isLatest bool

	// If current image not less than latest, is latest
	if !currentImage.LessThan(latestImageV) {
		isLatest = true
	}

	// If using the same image version, but the SHA has been updated upstream,
	// make not latest
	if currentImage.Equal(latestImageV) && currentSHA != latestImage.SHA && latestImage.SHA != "" {
		isLatest = false
		latestImage.Tag = fmt.Sprintf("%s@%s", latestImage.Tag, latestImage.SHA)
	}

	return latestImage, isLatest, nil
}

// isLatestSHA will return the the result of whether the given image is the latest, according to image SHA.
func (c *Checker) isLatestSHA(ctx context.Context, imageURL, currentSHA string, opts *api.Options) (*Result, error) {
	latestImage, err := c.search.LatestImage(ctx, imageURL, opts)
	if err != nil {
		return nil, err
	}

	isLatest := latestImage.SHA == currentSHA
	latestVersion := latestImage.SHA
	if len(latestImage.Tag) > 0 {
		latestVersion = fmt.Sprintf("%s@%s", latestImage.Tag, latestImage.SHA)
	}

	return &Result{
		CurrentVersion: currentSHA,
		LatestVersion:  latestVersion,
		IsLatest:       isLatest,
		ImageURL:       imageURL,
	}, nil
}

func (c *Checker) Search() search.Searcher {
	return c.search
}
