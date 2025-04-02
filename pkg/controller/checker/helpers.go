package checker

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

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
