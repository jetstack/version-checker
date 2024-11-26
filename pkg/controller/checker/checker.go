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
	search               search.Searcher
	imageURLSubstitution *Substitution
}

type Result struct {
	CurrentVersion string
	LatestVersion  string
	IsLatest       bool
	ImageURL       string
}

func New(search search.Searcher, imageURLSubstitution *Substitution) *Checker {
	return &Checker{
		search:               search,
		imageURLSubstitution: imageURLSubstitution,
	}
}

// Container will return the result of the given container's current version, compared to the latest upstream.
func (c *Checker) Container(ctx context.Context, log *logrus.Entry, pod *corev1.Pod,
	container *corev1.Container, opts *api.Options) (*Result, error) {
	statusSHA := containerStatusImageSHA(pod, container.Name)
	if len(statusSHA) == 0 {
		return nil, nil
	}

	imageURL, currentTag, currentSHA := urlTagSHAFromImage(container.Image)
	usingSHA, usingTag := len(currentSHA) > 0, len(currentTag) > 0

	if c.isLatestOrEmptyTag(currentTag) {
		c.handleLatestOrEmptyTag(log, currentTag, currentSHA, opts)
		usingTag = false
	}

	imageURL = c.substituteImageURL(log, imageURL)
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

func (c *Checker) substituteImageURL(log *logrus.Entry, imageURL string) string {

	if c.imageURLSubstitution == nil {
		return imageURL
	}
	newImageURL := c.imageURLSubstitution.Pattern.ReplaceAllString(imageURL, c.imageURLSubstitution.Substitute)
	if newImageURL != imageURL {
		log.Debugf("substituting image URL %s -> %s", imageURL, newImageURL)
	}
	return newImageURL
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

// containerStatusImageSHA will return the containers image SHA, if it is ready.
func containerStatusImageSHA(pod *corev1.Pod, containerName string) string {
	for _, status := range pod.Status.InitContainerStatuses {
		if status.Name == containerName {
			statusImage, _, statusSHA := urlTagSHAFromImage(status.ImageID)

			// If the image ID contains a URL, use the parsed SHA
			if len(statusSHA) > 0 {
				return statusSHA
			}

			return statusImage
		}
	}

	// Get the SHA of the current image
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == containerName {
			statusImage, _, statusSHA := urlTagSHAFromImage(status.ImageID)

			// If the image ID contains a URL, use the parsed SHA
			if len(statusSHA) > 0 {
				return statusSHA
			}

			return statusImage
		}
	}

	return ""
}

// isLatestOrEmptyTag will return true if the given tag is ” or 'latest'.
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

// urlTagSHAFromImage from will return the image URL, and the semver version
// and or SHA tag.
func urlTagSHAFromImage(image string) (url, version, sha string) {
	// If using SHA tag
	if split := strings.SplitN(image, "@", 2); len(split) > 1 {
		url = split[0]
		sha = split[1]

		// Check is url contains version, but also handle ports
		firstSlashIndex := strings.Index(split[0], "/")
		if firstSlashIndex == -1 {
			firstSlashIndex = 0
		}

		// url contains version
		if strings.LastIndex(split[0][firstSlashIndex:], ":") > -1 {
			lastColonIndex := strings.LastIndex(split[0], ":")
			url = split[0][:lastColonIndex]
			version = split[0][lastColonIndex+1:]
		}

		return
	}

	lastColonIndex := strings.LastIndex(image, ":")
	if lastColonIndex == -1 {
		return image, "", ""
	}

	if strings.LastIndex(image, "/") > lastColonIndex {
		return image, "", ""
	}

	return image[:lastColonIndex], image[lastColonIndex+1:], ""
}
