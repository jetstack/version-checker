package controller

import (
	"context"
	"fmt"
	"io"
	"net/url"
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

	return &ClusterVersionScheduler{
		log:      log,
		client:   kubernetes.NewForConfigOrDie(config),
		interval: interval,
		metrics:  metrics,
		channel:  channel,
	}
}

func (s *ClusterVersionScheduler) Start(ctx context.Context) error {
	go s.runScheduler(ctx)
	return s.reconcile(ctx)
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
			if err := s.reconcile(ctx); err != nil {
				s.log.Error(err, "Failed to reconcile cluster version")
			}
		}
	}
}

func (s *ClusterVersionScheduler) reconcile(_ context.Context) error {
	// Get current cluster version
	current, err := s.client.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("getting cluster version: %w", err)
	}

	// Get latest version from specified channel
	latest, err := getLatestVersion(s.channel)
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

func getLatestVersion(channel string) (string, error) {
	if !strings.HasSuffix(channel, ".txt") {
		channel += ".txt"
	}

	// Use url.JoinPath to safely join the base URL and channel path
	channelURL, err := url.JoinPath(channelURLSuffix, channel)
	if err != nil {
		return "", fmt.Errorf("failed to join channel URL: %w", err)
	}

	client := retryablehttp.NewClient()
	client.RetryMax = 3
	client.RetryWaitMin = 1 * time.Second
	client.RetryWaitMax = 30 * time.Second
	// Optional: Log using your own logrus/logr logger
	client.Logger = nil

	resp, err := client.Get(channelURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch from channel URL %s: %w", channelURL, err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", cerr)
		}
	}()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code %d when fetching channel %s", resp.StatusCode, channel)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	version := strings.TrimSpace(string(body))
	if version == "" {
		return "", fmt.Errorf("empty version returned from channel %s", channel)
	}

	return version, nil
}
