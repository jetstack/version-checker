package metrics

import (
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	m := NewServer(logrus.NewEntry(logrus.New()))

	// Lets add some Images/Metrics...
	for i, typ := range []string{"init", "container"} {
		version := fmt.Sprintf("0.1.%d", i)
		m.AddImage("namespace", "pod", "container", typ, "url", true, version, version)
	}

	// Check and ensure that the metrics are available...
	for i, typ := range []string{"init", "container"} {
		version := fmt.Sprintf("0.1.%d", i)
		mt, err := m.containerImageVersion.GetMetricWith(m.buildFullLabels("namespace", "pod", "container", typ, "url", version, version))
		require.NoError(t, err)
		count := testutil.ToFloat64(mt)
		require.Equal(t, count, float64(1))
	}

	// as well as the lastUpdated...
	for _, typ := range []string{"init", "container"} {
		mt, err := m.containerImageUpdated.GetMetricWith(m.buildLastUpdatedLabels("namespace", "pod", "container", typ, "url"))
		require.NoError(t, err)
		count := testutil.ToFloat64(mt)
		require.GreaterOrEqual(t, count, float64(time.Now().Unix()))
	}

	// Remove said metrics...
	for _, typ := range []string{"init", "container"} {
		m.RemoveImage("namespace", "pod", "container", typ)
	}
	// Ensure metrics and values return 0
	for i, typ := range []string{"init", "container"} {
		version := fmt.Sprintf("0.1.%d", i)
		mt, err := m.containerImageVersion.GetMetricWith(m.buildFullLabels("namespace", "pod", "container", typ, "url", version, version))
		require.NoError(t, err)
		count := testutil.ToFloat64(mt)
		require.Equal(t, count, float64(0))
	}
	// And the Last Updated is removed too
	for _, typ := range []string{"init", "container"} {
		mt, err := m.containerImageUpdated.GetMetricWith(m.buildLastUpdatedLabels("namespace", "pod", "container", typ, "url"))
		require.NoError(t, err)
		count := testutil.ToFloat64(mt)
		require.Equal(t, count, float64(0))
	}
}
