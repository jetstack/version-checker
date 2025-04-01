package controller

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/jetstack/version-checker/pkg/metrics"

	"github.com/Masterminds/semver/v3"
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

	// Get latest stable version
	latest, err := getLatestStableVersion(s.channel)
	if err != nil {
		return fmt.Errorf("fetching latest stable version: %w", err)
	}

	latestSemVer, err := semver.NewVersion(latest)
	if err != nil {
		return err
	}
	currentSemVer, err := semver.NewVersion(current.GitVersion)
	if err != nil {
		return err
	}
	// Strip metadata from the versions
	currentSemVerNoMeta, _ := currentSemVer.SetMetadata("")
	latestSemVerNoMeta, _ := latestSemVer.SetMetadata("")

	// Register metrics!
	s.metrics.RegisterKubeVersion(!currentSemVerNoMeta.LessThan(&latestSemVerNoMeta),
		currentSemVerNoMeta.String(), latestSemVerNoMeta.String(),
		s.channel,
	)

	s.log.WithFields(logrus.Fields{
		"currentVersion": currentSemVerNoMeta,
		"latestStable":   latestSemVerNoMeta,
		"channel":        s.channel,
	}).Info("Cluster version check complete")

	return nil
}

func getLatestStableVersion(channel string) (string, error) {
	if !strings.HasSuffix(channel, ".txt") {
		channel += ".txt"
	}

	// We don't need a `/` here as its should be in the channelURLSuffix
	channelURL := fmt.Sprintf("%s%s", channelURLSuffix, channel)

	client := retryablehttp.NewClient()
	client.RetryMax = 3
	client.RetryWaitMin = 1 * time.Second
	client.RetryWaitMax = 30 * time.Second
	// Optional: Log using your own logrus/logr logger
	client.Logger = nil

	resp, err := client.Get(channelURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}
