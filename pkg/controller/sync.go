package controller

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/joshvanl/version-checker/pkg/api"
	"github.com/joshvanl/version-checker/pkg/version/semver"
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

	// Check the image tag again after the cache timeout.
	c.workqueue.AddAfter(pod, c.cacheTimeout)

	if len(errs) > 0 {
		return fmt.Errorf("failed to sync pod %s/%s: %s",
			pod.Name, pod.Namespace, strings.Join(errs, ","))
	}

	return nil
}

// testContainerImage will test a given image version to the latest image
// available in the remote registry given the options.
func (c *Controller) testContainerImage(ctx context.Context, log *logrus.Entry,
	pod *corev1.Pod, container *corev1.Container, opts *api.Options) error {
	imageURL, currentTag := urlAndTagFromImage(container.Image)

	latestImage, err := c.getLatestImage(ctx, log, imageURL, opts)
	if err != nil {
		return err
	}

	var (
		latestTag string
		isLatest  bool
	)

	// if container is using latest or '' image tag, compare SHA tag
	if statusTag := currentTag; statusTag == "latest" ||
		statusTag == "" {
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == container.Name {
				_, currentTag = urlAndTagFromImage(status.ImageID)
				break
			}
		}

		if currentTag == "" {
			log.Errorf("image using %q tag, and image ID not yet set",
				statusTag)
			return nil
		}

		opts.UseSHA = true
		log.Warnf("image using %q tag, comparing image SHA %q",
			statusTag, currentTag)
	}

	if opts.UseSHA {
		// If we are using SHA then we can do a string comparison of the latest
		if currentTag == latestImage.SHA {
			isLatest = true
		}

		latestTag = latestImage.SHA
	} else {
		// Test against normal semvar
		currentImage := semver.Parse(currentTag)
		latestImageV := semver.Parse(latestImage.Tag)

		if !currentImage.LessThan(latestImageV) {
			isLatest = true
		}

		latestTag = latestImage.Tag
	}

	if isLatest {
		log.Debugf("image is latest %s:%s",
			imageURL, currentTag)
	} else {
		log.Debugf("image is not latest %s: %s -> %s",
			imageURL, currentTag, latestTag)
	}

	c.metrics.AddImage(pod.Namespace, pod.Name,
		container.Name, imageURL, currentTag, latestTag)

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

func urlAndTagFromImage(image string) (string, string) {
	imageSplit := strings.Split(image, "@")
	if len(imageSplit) == 2 {
		return imageSplit[0], imageSplit[1]
	}

	imageSplit = strings.Split(image, ":")
	if len(imageSplit) == 2 {
		return imageSplit[0], imageSplit[1]
	}

	return image, ""
}
