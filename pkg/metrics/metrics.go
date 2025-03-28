package metrics

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics is used to expose container image version checks as prometheus
// metrics.
type Metrics struct {
	*http.Server
	log *logrus.Entry

	registry               *prometheus.Registry
	containerImageVersion  *prometheus.GaugeVec
	containerImageDuration *prometheus.GaugeVec
	containerImageErrors   *prometheus.CounterVec

	// Contains all metrics for the roundtripper
	roundTripper *RoundTripper

	// container cache stores a cache of a container's current image, version,
	// and the latest
	containerCache map[string]cacheItem
	mu             sync.Mutex
}

type cacheItem struct {
	image          string
	currentVersion string
	latestVersion  string
}

func NewServer(log *logrus.Entry) *Metrics {
	// Reset the prometheus registry
	reg := prometheus.NewRegistry()

	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(collectors.NewGoCollector())

	containerImageVersion := promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "version_checker",
			Name:      "is_latest_version",
			Help:      "Where the container in use is using the latest upstream registry version",
		},
		[]string{
			"namespace", "pod", "container", "container_type", "image", "current_version", "latest_version",
		},
	)
	containerImageDuration := promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "version_checker",
			Name:      "image_lookup_duration",
			Help:      "Time taken to lookup version.",
		},
		[]string{"namespace", "pod", "container", "image"},
	)
	containerImageErrors := promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "version_checker",
			Name:      "image_failures_total",
			Help:      "Total number of errors where the version-checker was unable to get the latest upstream registry version",
		},
		[]string{
			"namespace", "pod", "container", "image",
		},
	)

	return &Metrics{
		log:                    log.WithField("module", "metrics"),
		registry:               reg,
		containerImageVersion:  containerImageVersion,
		containerImageDuration: containerImageDuration,
		containerImageErrors:   containerImageErrors,
		containerCache:         make(map[string]cacheItem),
		roundTripper:           NewRoundTripper(reg),
	}
}

// Run will run the metrics server.
func (m *Metrics) Run(servingAddress string) error {
	router := http.NewServeMux()
	router.Handle("/metrics", promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))
	router.Handle("/healthz", http.HandlerFunc(m.healthzAndReadyzHandler))
	router.Handle("/readyz", http.HandlerFunc(m.healthzAndReadyzHandler))

	ln, err := net.Listen("tcp", servingAddress)
	if err != nil {
		return err
	}

	m.Server = &http.Server{
		Addr:           ln.Addr().String(),
		ReadTimeout:    8 * time.Second,
		WriteTimeout:   8 * time.Second,
		MaxHeaderBytes: 1 << 15, // 1 MiB
		Handler:        router,
	}

	go func() {
		m.log.Infof("serving metrics on %s/metrics", ln.Addr())

		if err := m.Serve(ln); err != nil {
			m.log.Errorf("failed to serve prometheus metrics: %s", err)
			return
		}
	}()

	return nil
}

func (m *Metrics) AddImage(namespace, pod, container, containerType, imageURL string, isLatest bool, currentVersion, latestVersion string) {
	// Remove old image url/version if it exists
	m.RemoveImage(namespace, pod, container, containerType)

	m.mu.Lock()
	defer m.mu.Unlock()

	isLatestF := 0.0
	if isLatest {
		isLatestF = 1.0
	}

	m.containerImageVersion.With(
		m.buildLabels(namespace, pod, container, containerType, imageURL, currentVersion, latestVersion),
	).Set(isLatestF)

	index := m.latestImageIndex(namespace, pod, container, containerType)
	m.containerCache[index] = cacheItem{
		image:          imageURL,
		currentVersion: currentVersion,
		latestVersion:  latestVersion,
	}
}

func (m *Metrics) RemoveImage(namespace, pod, container, containerType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	index := m.latestImageIndex(namespace, pod, container, containerType)
	_, ok := m.containerCache[index]
	if !ok {
		return
	}

	m.containerImageVersion.DeletePartialMatch(
		m.buildPartialLabels(namespace, pod),
	)
	m.containerImageDuration.DeletePartialMatch(
		m.buildPartialLabels(namespace, pod),
	)
	delete(m.containerCache, index)
}

func (m *Metrics) RegisterImageDuration(namespace, pod, container, image string, startTime time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.containerImageDuration.WithLabelValues(namespace, pod, container, image).
		Set(time.Since(startTime).Seconds())
}

func (m *Metrics) latestImageIndex(namespace, pod, container, containerType string) string {
	return strings.Join([]string{namespace, pod, container, containerType}, "")
}

func (m *Metrics) ErrorsReporting(namespace, pod, container, imageURL string) {

	m.containerImageErrors.WithLabelValues(namespace, pod, container, imageURL).Inc()
}

func (m *Metrics) buildLabels(namespace, pod, container, containerType, imageURL, currentVersion, latestVersion string) prometheus.Labels {
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

func (m *Metrics) buildPartialLabels(namespace, pod string) prometheus.Labels {
	return prometheus.Labels{
		"namespace": namespace,
		"pod":       pod,
	}
}

func (m *Metrics) Shutdown() error {
	// If metrics server is not started than exit early
	if m.Server == nil {
		return nil
	}

	m.log.Info("shutting down prometheus metrics server...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := m.Server.Shutdown(ctx); err != nil {
		return fmt.Errorf("prometheus metrics server shutdown failed: %s", err)
	}

	m.log.Info("prometheus metrics server gracefully stopped")

	return nil
}

func (m *Metrics) healthzAndReadyzHandler(w http.ResponseWriter, _ *http.Request) {
	// Its not great, but does help ensure that we're alive and ready over
	// calling the /metrics endpoint which can be expensive on large payloads
	_, err := w.Write([]byte("OK"))
	if err != nil {
		m.log.Errorf("Failed to send Healthz/Readyz response: %s", err)
	}
}
