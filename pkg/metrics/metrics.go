package metrics

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
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

	latestImageLabel map[string]string
}

func New(log *logrus.Entry) *Metrics {
	containerImageVersion := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "version_checker",
			Name:      "is_latest_version",
			Help:      "Where the container in use is using the latest upstream registry version",
		},
		[]string{
			"namespace", "pod", "container", "image", "current_version", "latest_version",
		},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(containerImageVersion)

	return &Metrics{
		log:                   log.WithField("module", "metrics"),
		registry:              registry,
		containerImageVersion: containerImageVersion,
		latestImageLabel:      make(map[string]string),
	}
}

// Run will run the metrics server
func (m *Metrics) Run(servingAddress string) error {
	router := http.NewServeMux()
	router.Handle("/metrics", promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))

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

func (m *Metrics) AddImage(namespace, pod, container, imageURL string, currentImage, latestImage string) {
	isLatest := 0.0
	if currentImage == latestImage {
		isLatest = 1.0
	}

	m.containerImageVersion.With(
		m.buildLabels(namespace, pod, container, imageURL, currentImage, latestImage),
	).Set(isLatest)

	index := m.latestImageIndex(namespace, pod, container)
	m.latestImageLabel[index] = latestImage
}

func (m *Metrics) RemoveImage(namespace, pod, container, imageURL, currentImage string) {
	index := m.latestImageIndex(namespace, pod, container)
	m.containerImageVersion.Delete(
		m.buildLabels(namespace, pod, container, imageURL, currentImage,
			m.latestImageLabel[index],
		),
	)
	delete(m.latestImageLabel, index)
}

func (m *Metrics) latestImageIndex(namespace, pod, container string) string {
	return strings.Join([]string{namespace, pod, container}, "")
}

func (m *Metrics) buildLabels(namespace, pod, container, imageURL, currentImage, latestImage string) prometheus.Labels {
	return prometheus.Labels{
		"namespace":       namespace,
		"pod":             pod,
		"container":       container,
		"image":           imageURL,
		"current_version": currentImage,
		"latest_version":  latestImage,
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
