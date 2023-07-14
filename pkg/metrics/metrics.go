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
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics is used to expose container image version checks as prometheus
// metrics.
type Metrics struct {
	*http.Server

	registry              *prometheus.Registry
	containerImageVersion *prometheus.GaugeVec
	log                   *logrus.Entry

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

func New(log *logrus.Entry) *Metrics {
	containerImageVersion := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "version_checker",
			Name:      "is_latest_version",
			Help:      "Where the container in use is using the latest upstream registry version",
		},
		[]string{
			"namespace", "pod", "container", "container_type", "image", "current_version", "latest_version",
		},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(containerImageVersion)

	return &Metrics{
		log:                   log.WithField("module", "metrics"),
		registry:              registry,
		containerImageVersion: containerImageVersion,
		containerCache:        make(map[string]cacheItem),
	}
}

// Run will run the metrics server
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

	index := m.latestImageIndex(namespace, pod, container)
	m.containerCache[index] = cacheItem{
		image:          imageURL,
		currentVersion: currentVersion,
		latestVersion:  latestVersion,
	}
}

func (m *Metrics) RemoveImage(namespace, pod, container, containerType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	index := m.latestImageIndex(namespace, pod, container)
	item, ok := m.containerCache[index]
	if !ok {
		return
	}

	m.containerImageVersion.Delete(
		m.buildLabels(namespace, pod, container, containerType, item.image, item.currentVersion, item.latestVersion),
	)
	delete(m.containerCache, index)
}

func (m *Metrics) latestImageIndex(namespace, pod, container string) string {
	return strings.Join([]string{namespace, pod, container}, "")
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

func (m *Metrics) healthzAndReadyzHandler(w http.ResponseWriter, r *http.Request) {
	// Its not great, but does help ensure that we're alive and ready over
	// calling the /metrics endpoint which can be expensive on large payloads
	w.Write([]byte("OK"))
}
