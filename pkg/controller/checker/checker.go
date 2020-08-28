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

// Container will return the result of the given container's current version, compared to the latest upstream
func (c *Checker) Container(ctx context.Context, log *logrus.Entry,
	pod *corev1.Pod, container *corev1.Container, opts *api.Options) (*Result, error) {

	// If the container image SHA status is not ready yet, exit early
	statusSHA := containerStatusImageSHA(pod, container.Name)
	if len(statusSHA) == 0 {
		return nil, nil
	}

	imageURL, currentTag, currentSHA := urlTagSHAFromImage(container.Image)

	usingSHA := len(currentSHA) > 0
	usingTag := len(currentTag) > 0

	// If using latest or no tag, then compare on SHA
	if c.isLatestOrEmptyTag(currentTag) {
		// Override options to use SHA
		opts.UseSHA = true
		usingTag = false
		log.WithField("module", "checker").Debugf("image using %q tag, comparing image SHA %q",
			currentTag, currentSHA)
	}

	if opts.UseSHA {
		result, err := c.isLatestSHA(ctx, imageURL, statusSHA, opts)
		if err != nil {
			return nil, err
		}

		if usingTag {
			result.CurrentVersion = fmt.Sprintf("%s@%s", currentTag, result.CurrentVersion)
		}

		return result, nil
	}

	currentImage := semver.Parse(currentTag)
	latestImage, isLatest, err := c.isLatestSemver(ctx, imageURL, statusSHA, currentImage, opts)
	if err != nil {
		return nil, err
	}

	latestVersion := latestImage.Tag

	// If we are using SHA and tag, make latest version include both
	if usingSHA && !strings.Contains(latestVersion, "@") {
		latestVersion = fmt.Sprintf("%s@%s", latestVersion, latestImage.SHA)
	}

	// If latest version contains SHA, include in current version
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

// containerStatusImageSHA will return the containers image SHA, if it is ready
func containerStatusImageSHA(pod *corev1.Pod, containerName string) string {
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

// isLatestOrEmptyTag will return true if the given tag is '' or 'latest'
func (c *Checker) isLatestOrEmptyTag(tag string) bool {
	return tag == "" || tag == "latest"
}

// isLatestSemver will return the latest image, and whether the given image is the latest
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
	if currentImage.Equal(latestImageV) && currentSHA != latestImage.SHA {
		isLatest = false
		latestImage.Tag = fmt.Sprintf("%s@%s", latestImage.Tag, latestImage.SHA)
	}

	return latestImage, isLatest, nil
}

// isLatestSHA will return the the result of whether the given image is the latest, according to image SHA
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
