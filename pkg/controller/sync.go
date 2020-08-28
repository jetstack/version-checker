package controller

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/jetstack/version-checker/pkg/api"
	versionerrors "github.com/jetstack/version-checker/pkg/version/errors"
	"github.com/jetstack/version-checker/pkg/version/semver"
)

// sync will enqueue a given pod to run against the version checker.
func (c *Controller) sync(ctx context.Context, pod *corev1.Pod) error {
	log := c.log.WithField("name", pod.Name).WithField("namespace", pod.Namespace)

	var errs []string
	for _, container := range pod.Spec.Containers {
		enable, ok := pod.Annotations[api.EnableAnnotationKey+"/"+container.Name]
		if c.defaultTestAll {
			// If default all and we explicitly disable, ignore
			if ok && enable == "false" {
				continue
			}
		} else {
			// If not default all and we don't enable, ignore
			if !ok || enable != "true" {
				continue
			}
		}

		log = log.WithField("container", container.Name)
		log.Debug("processing conainer image")

		opts, err := c.buildOptions(container.Name, pod.Annotations)
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to build options from annotations for %q: %s",
				container.Name, err))
			continue
		}

		if err := c.testContainerImage(ctx, log, pod, &container, opts); err != nil {
			errs = append(errs, fmt.Sprintf("failed to test container image %q: %s",
				container.Name, err))
			continue
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to sync pod %s/%s: %s",
			pod.Name, pod.Namespace, strings.Join(errs, ","))
	}

	return nil
}

// testContainerImage will test a given image version to the latest image
// available in the remote registry, given the options.
func (c *Controller) testContainerImage(ctx context.Context, log *logrus.Entry,
	pod *corev1.Pod, container *corev1.Container, opts *api.Options) error {

	imageURL, currentTag, currentSHA := urlTagSHAFromImage(container.Image)
	usingSHA := len(currentSHA) > 0

	currentMetricsVersion := metricsLabel(currentTag, currentSHA)

	if !usingSHA {
		// Get the SHA of the current image
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == container.Name {
				statusImage, _, statusSHA := urlTagSHAFromImage(status.ImageID)
				currentSHA = statusImage

				// If the image ID contains a URL, use the parsed SHA
				if len(statusSHA) > 0 {
					currentSHA = statusSHA
				}

				break
			}
		}
	}

	// if no image SHA set on container yet and none specified, wait for next
	// sync
	if currentSHA == "" {
		return nil
	}

	// if tag is set to latest or "", use latest SHA comparison
	if currentTag == "" || currentTag == "latest" {
		opts.UseSHA = true
		currentTag = currentSHA

		log.Debugf("image using %q tag, comparing image SHA %q",
			currentTag, currentSHA)
	}

	latestImage, err := c.search.LatestImage(ctx, log, imageURL, opts)
	// Don't re-sync, if no version found meeting search criteria
	if versionerrors.IsNoVersionFound(err) {
		log.Error(err.Error())
		return nil
	}
	if err != nil {
		return err
	}

	var (
		latestVersion string
		isLatest      bool
	)

	if opts.UseSHA {
		currentTag = currentSHA
		latestVersion = latestImage.SHA

		// If we are using SHA, then we can do a string comparison of the latest
		if currentTag == latestImage.SHA {
			isLatest = true
		}
	} else {
		// Test against normal semvar
		currentImage := semver.Parse(currentTag)
		latestImageV := semver.Parse(latestImage.Tag)

		// If current image not less than latest, is latest
		if !currentImage.LessThan(latestImageV) {
			isLatest = true
		}

		// If using the same image version, but the SHA has been updated upstream,
		// make not latest
		if currentImage.Equal(latestImageV) && currentSHA != latestImage.SHA {
			isLatest = false
		}

		latestVersion = latestImage.Tag

		// If we are using SHA and tag, make latest version include both
		if usingSHA {
			latestVersion = fmt.Sprintf("%s@%s", latestVersion, latestImage.SHA)
		}
	}

	if isLatest {
		log.Debugf("image is latest %s:%s",
			imageURL, currentMetricsVersion)
	} else {
		log.Debugf("image is not latest %s: %s -> %s",
			imageURL, currentMetricsVersion, latestVersion)
	}

	c.metrics.AddImage(pod.Namespace, pod.Name,
		container.Name, imageURL, currentMetricsVersion, latestVersion)

	return nil
}

// buildOptions will build the tag options based on pod annotations.
func (c *Controller) buildOptions(containerName string, annotations map[string]string) (*api.Options, error) {
	var (
		opts      api.Options
		errs      []string
		setNonSha bool
	)

	if useSHA, ok := annotations[api.UseSHAAnnotationKey+"/"+containerName]; ok && useSHA == "true" {
		opts.UseSHA = true
	}

	if useMetaData, ok := annotations[api.UseMetaDataAnnotationKey+"/"+containerName]; ok && useMetaData == "true" {
		setNonSha = true
		opts.UseMetaData = true
	}

	if matchRegex, ok := annotations[api.MatchRegexAnnotationKey+"/"+containerName]; ok {
		setNonSha = true
		opts.MatchRegex = &matchRegex

		regexMatcher, err := regexp.Compile(matchRegex)
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to compile regex at annotation %q: %s",
				api.MatchRegexAnnotationKey, err))
		} else {
			opts.RegexMatcher = regexMatcher
		}
	}

	if pinMajor, ok := annotations[api.PinMajorAnnotationKey+"/"+containerName]; ok {
		setNonSha = true

		ma, err := strconv.ParseInt(pinMajor, 10, 64)
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to parse %s: %s",
				api.PinMajorAnnotationKey+"/"+containerName, err))
		} else {
			opts.PinMajor = &ma
		}
	}

	if pinMinor, ok := annotations[api.PinMinorAnnotationKey+"/"+containerName]; ok {
		setNonSha = true

		if opts.PinMajor == nil {
			errs = append(errs, fmt.Sprintf("unable to set %q without setting %q",
				api.PinMinorAnnotationKey+"/"+containerName, api.PinMajorAnnotationKey+"/"+containerName))
		} else {

			mi, err := strconv.ParseInt(pinMinor, 10, 64)
			if err != nil {
				errs = append(errs, fmt.Sprintf("failed to parse %s: %s",
					api.PinMinorAnnotationKey+"/"+containerName, err))
			} else {
				opts.PinMinor = &mi
			}
		}
	}

	if pinPatch, ok := annotations[api.PinPatchAnnotationKey+"/"+containerName]; ok {
		setNonSha = true

		if opts.PinMajor == nil && opts.PinMinor == nil {
			errs = append(errs, fmt.Sprintf("unable to set %q without setting %q or %q",
				api.PinPatchAnnotationKey+"/"+containerName,
				api.PinMinorAnnotationKey+"/"+containerName,
				api.PinMajorAnnotationKey+"/"+containerName))
		} else {

			pa, err := strconv.ParseInt(pinPatch, 10, 64)
			if err != nil {
				errs = append(errs, fmt.Sprintf("failed to parse %s: %s",
					api.PinPatchAnnotationKey+"/"+containerName, err))
			} else {
				opts.PinPatch = &pa
			}
		}
	}

	if opts.UseSHA && setNonSha {
		errs = append(errs, fmt.Sprintf("cannot define %q with any semver otions",
			api.UseSHAAnnotationKey+"/"+containerName))
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to build version options: %s",
			strings.Join(errs, ", "))
	}

	return &opts, nil
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

	if split := strings.Split(image, ":"); len(split) == 2 {
		return split[0], split[1], ""
	}

	return image, "", ""
}

// metricsLabel will return a version string, containing the tag and/or sha
func metricsLabel(tag, sha string) string {
	if len(sha) > 0 {
		if len(tag) == 0 {
			tag = sha
		} else {
			tag = fmt.Sprintf("%s@%s", tag, sha)
		}
	}

	return tag
}
