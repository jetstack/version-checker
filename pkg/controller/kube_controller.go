package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/jetstack/version-checker/pkg/metrics"
	"github.com/jetstack/version-checker/pkg/version/semver"
)

const channelURLSuffix = "https://dl.k8s.io/release/"

type ClusterVersionScheduler struct {
	client   kubernetes.Interface
	http     http.Client
	log      *logrus.Entry
	metrics  *metrics.Metrics
	interval time.Duration
	channel  string
}

func NewKubeReconciler(
	log *logrus.Entry,
	config *rest.Config,
	metrics *metrics.Metrics,
	interval time.Duration,
	channel string,
) *ClusterVersionScheduler {
	// If no channel is specified, return nil to indicate disabled
	if channel == "" {
		log.Info("Kubernetes version checking disabled (no channel specified)")
		return nil
	}
	log = log.WithField("controller", "channel")

	httpClient := retryablehttp.NewClient()
	httpClient.RetryMax = 3
	httpClient.RetryWaitMin = 1 * time.Second
	httpClient.RetryWaitMax = 30 * time.Second
	httpClient.Logger = log

	return &ClusterVersionScheduler{
		log:      log.WithField("channel", channel),
		client:   kubernetes.NewForConfigOrDie(config),
		http:     *httpClient.StandardClient(),
		interval: interval,
		metrics:  metrics,
		channel:  channel,
	}
}

func (s *ClusterVersionScheduler) Start(ctx context.Context) error {
	go s.runScheduler(ctx)
	// Run an initial check on startup
	return s.reconcile()
}

func (s *ClusterVersionScheduler) runScheduler(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.log.WithField("interval", s.interval).WithField("channel", s.channel).
		Info("ClusterVersionScheduler started")

	for {
		select {
		case <-ctx.Done():
			s.log.Info("ClusterVersionScheduler stopping")
			return
		case <-ticker.C:
			if err := s.reconcile(); err != nil {
				s.log.Error(err, "Failed to reconcile cluster version")
			}
		}
	}
}

func (s *ClusterVersionScheduler) reconcile() error {
	// Get current cluster version
	current, err := s.client.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("getting cluster version: %w", err)
	}

	// Get latest version from specified channel
	latest, err := s.getLatestVersion(s.channel)
	if err != nil {
		return fmt.Errorf("fetching latest version from channel %s: %w", s.channel, err)
	}

	latestSemVer := semver.Parse(latest)
	currentSemVer := semver.Parse(current.GitVersion)

	// Create version strings without metadata for comparison
	currentSemVerNoMeta := fmt.Sprintf("%d.%d.%d", currentSemVer.Major(), currentSemVer.Minor(), currentSemVer.Patch())
	latestSemVerNoMeta := fmt.Sprintf("%d.%d.%d", latestSemVer.Major(), latestSemVer.Minor(), latestSemVer.Patch())

	// Parse the versions without metadata for comparison
	currentComparable := semver.Parse(currentSemVerNoMeta)
	latestComparable := semver.Parse(latestSemVerNoMeta)

	// Register metrics!
	s.metrics.RegisterKubeVersion(!currentComparable.LessThan(latestComparable),
		currentSemVerNoMeta, latestSemVerNoMeta,
		s.channel,
	)

	s.log.WithFields(logrus.Fields{
		"currentVersion": currentSemVerNoMeta,
		"latestVersion":  latestSemVerNoMeta,
		"channel":        s.channel,
	}).Info("Cluster version check complete")

	return nil
}

func (s *ClusterVersionScheduler) getLatestVersion(channel string) (string, error) {
	// Always use upstream Kubernetes channels - this is the authoritative source
	// Platform detection is kept for logging purposes only
	return s.getLatestVersionFromUpstream(channel)
}

func (s *ClusterVersionScheduler) getLatestVersionFromUpstream(channel string) (string, error) {
	// Validate channel - only allow known Kubernetes channels
	if !isValidKubernetesChannel(channel) {
		return "", fmt.Errorf("unsupported channel: %s. Valid channels: stable, latest, latest-1.xx", channel)
	}

	if !strings.HasSuffix(channel, ".txt") {
		channel += ".txt"
	}

	channelURL, err := url.JoinPath(channelURLSuffix, channel)
	if err != nil {
		return "", fmt.Errorf("failed to join channel URL: %w", err)
	}

	resp, err := s.http.Get(channelURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch from channel URL %s: %w", channelURL, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d when fetching channel %s", resp.StatusCode, channel)
	}
	if resp.Header.Get("content-type") != "text/plain" {
		return "", fmt.Errorf("unexpected content-type %s when fetching channel %s", resp.Header.Get("content-type"), channel)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	version := strings.TrimSpace(string(body))
	if version == "" {
		return "", fmt.Errorf("empty version received from channel %s", channel)
	}

	return version, nil
}

func isValidKubernetesChannel(channel string) bool {
	// Only allow official Kubernetes channels
	validChannels := []string{"stable", "latest"}

	// Allow latest-X.Y and stable-X.Y formats
	if strings.HasPrefix(channel, "latest-1.") || strings.HasPrefix(channel, "stable-1.") {
		return true
	}

	return slices.Contains(validChannels, channel)
}
