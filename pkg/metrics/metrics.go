package metrics

import (
	"context"
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

	registry               ctrmetrics.RegistererGatherer
	containerImageVersion  *prometheus.GaugeVec
	containerImageChecked  *prometheus.GaugeVec
	containerImageDuration *prometheus.GaugeVec
	containerImageErrors   *prometheus.CounterVec

	cache k8sclient.Reader

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

	return &Metrics{
		log:   log.WithField("module", "metrics"),
		cache: cache,

		registry:               reg,
		containerImageVersion:  containerImageVersion,
		containerImageDuration: containerImageDuration,
		containerImageChecked:  containerImageChecked,
		containerImageErrors:   containerImageErrors,
		roundTripper:           NewRoundTripper(reg),
	}
}

func (m *Metrics) AddImage(namespace, pod, container, containerType, imageURL string, isLatest bool, currentVersion, latestVersion string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	isLatestF := 0.0
	if isLatest {
		isLatestF = 1.0
	}

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
	total += m.containerImageDuration.DeletePartialMatch(labels)
	total += m.containerImageChecked.DeletePartialMatch(labels)
	total += m.containerImageErrors.DeletePartialMatch(labels)

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

	m.log.Infof("Removed %d metrics for pod %s/%s", total, namespace, pod)
}

func (m *Metrics) RegisterImageDuration(namespace, pod, container, image string, startTime time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.PodExists(context.Background(), namespace, pod) {
		m.log.WithField("metric", "RegisterImageDuration").Warnf("pod %s/%s not found, not registering error", namespace, pod)
		return
	}

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

	m.containerImageErrors.WithLabelValues(
		namespace, pod, container, imageURL,
	).Inc()
}
