package metrics

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/prometheus/client_golang/prometheus"
)

func buildFullLabels(namespace, pod, container, containerType, imageURL, currentVersion, latestVersion string) prometheus.Labels {
	return prometheus.Labels{
		"namespace":       namespace,
		"pod":             pod,
		"container_type":  containerType,
		"container":       container,
		"image":           imageURL,
		"current_version": currentVersion,
		"latest_version":  latestVersion,
	}
}

func buildLastUpdatedLabels(namespace, pod, container, containerType, imageURL string) prometheus.Labels {
	return prometheus.Labels{
		"namespace":      namespace,
		"pod":            pod,
		"container_type": containerType,
		"container":      container,
		"image":          imageURL,
	}
}

// buildContainerImageLabels matches the metrics that are keyed by container
// and image but have no container_type label.
func buildContainerImageLabels(namespace, pod, container string) prometheus.Labels {
	return prometheus.Labels{"namespace": namespace, "pod": pod, "container": container}
}

func buildPodPartialLabels(namespace, pod string) prometheus.Labels {
	return prometheus.Labels{
		"namespace": namespace,
		"pod":       pod,
	}
}

func buildContainerPartialLabels(namespace, pod, container, containerType string) prometheus.Labels {
	return prometheus.Labels{
		"namespace":      namespace,
		"pod":            pod,
		"container":      container,
		"container_type": containerType,
	}
}

// This _should_ leverage the Controllers Cache
func (m *Metrics) PodExists(ctx context.Context, ns, name string) bool {
	pod := &corev1.Pod{}
	err := m.cache.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, pod)
	return err == nil && pod.GetDeletionTimestamp() == nil
}
