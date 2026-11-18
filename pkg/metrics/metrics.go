package metrics

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ctrmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

const MetricNamespace = "version_checker"

// Metrics is used to expose container image version checks as prometheus
// metrics.
type Metrics struct {
	log *logrus.Entry

	registry                ctrmetrics.RegistererGatherer
	containerImageVersion   *prometheus.GaugeVec
	containerImageChecked   *prometheus.GaugeVec
	containerImageDuration  *prometheus.GaugeVec
	containerImageErrors    *prometheus.CounterVec
	containerImageTimestamp *prometheus.GaugeVec
	containerImageAvailable *prometheus.GaugeVec

	// Kubernetes version metric
	kubernetesVersion *prometheus.GaugeVec

	cache k8sclient.Reader

	// errorImage remembers which image each container's failure counter is
	// for (key "namespace/pod/container"), so the counter of a replaced image
	// can be dropped instead of accumulating series.
	errorImage map[string]string

	// Contains all metrics for the roundtripper
	roundTripper *RoundTripper

	mu sync.Mutex
}

// func New(log *logrus.Entry, reg ctrmetrics.RegistererGatherer, kubeClient k8sclient.Client) *Metrics {
func New(log *logrus.Entry, reg ctrmetrics.RegistererGatherer, cache k8sclient.Reader) *Metrics {
	// Attempt to register, but ignore errors
	// TODO: We should check for AlreadyRegisteredError err type here for better error handling
	_ = reg.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	_ = reg.Register(collectors.NewGoCollector())

	containerImageVersion := promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: MetricNamespace,
			Name:      "is_latest_version",
			Help:      "Where the container in use is using the latest upstream registry version",
		},
		[]string{
			"namespace", "pod", "container", "container_type", "image", "current_version", "latest_version",
		},
	)
	containerImageChecked := promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: MetricNamespace,
			Name:      "last_checked",
			Help:      "Timestamp when the image was checked",
		},
		[]string{
			"namespace", "pod", "container", "container_type", "image",
		},
	)
	containerImageDuration := promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: MetricNamespace,
			Name:      "image_lookup_duration",
			Help:      "Time taken to lookup version.",
		},
		[]string{"namespace", "pod", "container", "image"},
	)
	containerImageErrors := promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: MetricNamespace,
			Name:      "image_failures_total",
			Help:      "Total number of errors where the version-checker was unable to get the latest upstream registry version",
		},
		[]string{
			"namespace", "pod", "container", "image",
		},
	)
	kubernetesVersion := promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "version_checker",
			Name:      "is_latest_kube_version",
			Help:      "Where the current cluster is using the latest release channel version",
		},
		[]string{
			"current_version", "latest_version", "channel",
		},
	)
	containerImageTimestamp := promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: MetricNamespace,
			Name:      "image_timestamp",
			Help:      "Creation timestamp (unix seconds) of the currently running image",
		},
		[]string{
			"namespace", "pod", "container", "container_type", "image",
		},
	)
	containerImageAvailable := promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: MetricNamespace,
			Name:      "is_available",
			Help:      "Whether the currently running image was found upstream (1) or no longer exists (0)",
		},
		[]string{
			"namespace", "pod", "container", "container_type", "image",
		},
	)

	return &Metrics{
		log:        log.WithField("module", "metrics"),
		cache:      cache,
		errorImage: map[string]string{},

		registry:                reg,
		containerImageVersion:   containerImageVersion,
		containerImageDuration:  containerImageDuration,
		containerImageChecked:   containerImageChecked,
		containerImageErrors:    containerImageErrors,
		kubernetesVersion:       kubernetesVersion,
		containerImageTimestamp: containerImageTimestamp,
		containerImageAvailable: containerImageAvailable,
		roundTripper:            NewRoundTripper(reg),
	}
}

func (m *Metrics) AddImage(namespace, pod, container, containerType, imageURL string, isLatest bool, currentVersion, latestVersion string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	isLatestF := 0.0
	if isLatest {
		isLatestF = 1.0
	}

	labels := buildContainerPartialLabels(namespace, pod, container, containerType)

	// Remove any existing "current state" gauge series for this container before
	// registering the newest values. Otherwise each version change leaves behind
	// a distinct Prometheus series due to the current/latest version labels.
	m.containerImageVersion.DeletePartialMatch(labels)
	m.containerImageChecked.DeletePartialMatch(labels)

	m.containerImageVersion.With(
		buildFullLabels(namespace, pod, container, containerType, imageURL, currentVersion, latestVersion),
	).Set(isLatestF)

	// Bump last updated timestamp
	m.containerImageChecked.With(
		buildLastUpdatedLabels(namespace, pod, container, containerType, imageURL),
	).Set(float64(time.Now().Unix()))
}

func (m *Metrics) RemoveImage(namespace, pod, container, containerType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	total := 0

	labels := buildContainerPartialLabels(namespace, pod, container, containerType)

	total += m.containerImageVersion.DeletePartialMatch(labels)
	total += m.containerImageChecked.DeletePartialMatch(labels)
	// Duration and errors have no container_type label, so a partial match
	// that includes it would match nothing.
	total += m.containerImageDuration.DeletePartialMatch(buildContainerImageLabels(namespace, pod, container))
	total += m.containerImageErrors.DeletePartialMatch(buildContainerImageLabels(namespace, pod, container))
	delete(m.errorImage, namespace+"/"+pod+"/"+container)
	total += m.containerImageTimestamp.DeletePartialMatch(labels)
	total += m.containerImageAvailable.DeletePartialMatch(labels)

	m.log.Infof("Removed %d metrics for image %s/%s/%s (%s)", total, namespace, pod, container, containerType)
}

func (m *Metrics) RemovePod(namespace, pod string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	total := 0
	total += m.containerImageVersion.DeletePartialMatch(
		buildPodPartialLabels(namespace, pod),
	)
	total += m.containerImageDuration.DeletePartialMatch(
		buildPodPartialLabels(namespace, pod),
	)
	total += m.containerImageChecked.DeletePartialMatch(
		buildPodPartialLabels(namespace, pod),
	)
	total += m.containerImageErrors.DeletePartialMatch(
		buildPodPartialLabels(namespace, pod),
	)
	for k := range m.errorImage {
		if strings.HasPrefix(k, namespace+"/"+pod+"/") {
			delete(m.errorImage, k)
		}
	}
	total += m.containerImageTimestamp.DeletePartialMatch(
		buildPodPartialLabels(namespace, pod),
	)
	total += m.containerImageAvailable.DeletePartialMatch(
		buildPodPartialLabels(namespace, pod),
	)

	m.log.Infof("Removed %d metrics for pod %s/%s", total, namespace, pod)
}

func (m *Metrics) RegisterImageDuration(namespace, pod, container, image string, startTime time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.PodExists(context.Background(), namespace, pod) {
		m.log.WithField("metric", "RegisterImageDuration").Warnf("pod %s/%s not found, not registering error", namespace, pod)
		return
	}

	// One series per container: drop the one for a previous image.
	m.containerImageDuration.DeletePartialMatch(buildContainerImageLabels(namespace, pod, container))
	m.containerImageDuration.WithLabelValues(
		namespace, pod, container, image,
	).Set(time.Since(startTime).Seconds())
}

func (m *Metrics) ReportError(namespace, pod, container, imageURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.PodExists(context.Background(), namespace, pod) {
		m.log.WithField("metric", "ReportError").Warnf("pod %s/%s not found, not registering error", namespace, pod)
		return
	}

	// A replaced image starts a new failure count rather than a new series
	// next to the old one.
	key := namespace + "/" + pod + "/" + container
	if prev, ok := m.errorImage[key]; ok && prev != imageURL {
		m.containerImageErrors.DeletePartialMatch(buildContainerImageLabels(namespace, pod, container))
	}
	m.errorImage[key] = imageURL

	m.containerImageErrors.WithLabelValues(
		namespace, pod, container, imageURL,
	).Inc()
}

func (m *Metrics) ImageTimestamp(namespace, pod, container, containerType, imageURL string, timestamp time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.PodExists(context.Background(), namespace, pod) {
		m.log.WithField("metric", "ImageTimestamp").Warnf("pod %s/%s not found, not registering timestamp", namespace, pod)
		return
	}

	// Ensure we don't leave a stale timestamp series behind if the image label
	// changes or if the registry stops reporting a valid creation time.
	m.containerImageTimestamp.DeletePartialMatch(
		buildContainerPartialLabels(namespace, pod, container, containerType),
	)

	// An unpopulated time.Time has a huge negative Unix() value, and some
	// registry clients report an epoch-0 (1970) timestamp when they cannot
	// determine a creation time. Both would be garbage in Prometheus (and read
	// as "infinitely old"), so only record strictly-positive timestamps.
	if timestamp.IsZero() || timestamp.Unix() <= 0 {
		return
	}

	m.containerImageTimestamp.With(
		buildLastUpdatedLabels(namespace, pod, container, containerType, imageURL),
	).Set(float64(timestamp.Unix()))
}

func (m *Metrics) ImageAvailable(namespace, pod, container, containerType, imageURL string, available bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.PodExists(context.Background(), namespace, pod) {
		m.log.WithField("metric", "ImageAvailable").Warnf("pod %s/%s not found, not registering availability", namespace, pod)
		return
	}

	// Ensure we don't leave stale series behind if the image label changes.
	m.containerImageAvailable.DeletePartialMatch(
		buildContainerPartialLabels(namespace, pod, container, containerType),
	)

	value := 0.0
	if available {
		value = 1.0
	}

	m.containerImageAvailable.With(
		buildLastUpdatedLabels(namespace, pod, container, containerType, imageURL),
	).Set(value)
}
