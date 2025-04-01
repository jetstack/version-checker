package metrics

import "github.com/prometheus/client_golang/prometheus"

func (m *Metrics) RegisterKubeVersion(isLatest bool, currentVersion, latestVersion, channel string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	isLatestF := 0.0
	if isLatest {
		isLatestF = 1.0
	}

	m.kubernetesVersion.With(
		prometheus.Labels{
			"current_version": currentVersion,
			"latest_version":  latestVersion,
			"channel":         channel,
		},
	).Set(isLatestF)
}
