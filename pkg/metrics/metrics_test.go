package metrics

import (
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
)

func TestCache(t *testing.T) {
	m := NewServer(logrus.NewEntry(logrus.New()))

	for i, typ := range []string{"init", "container"} {
		version := fmt.Sprintf("0.1.%d", i)
		m.AddImage("namespace", "pod", "container", typ, "url", true, version, version)
	}

	for i, typ := range []string{"init", "container"} {
		version := fmt.Sprintf("0.1.%d", i)
		mt, _ := m.containerImageVersion.GetMetricWith(m.buildLabels("namespace", "pod", "container", typ, "url", version, version))
		count := testutil.ToFloat64(mt)
		if count != 1 {
			t.Error("Should have added metric")
		}
	}

	for _, typ := range []string{"init", "container"} {
		m.RemoveImage("namespace", "pod", "container", typ)
	}
	for i, typ := range []string{"init", "container"} {
		version := fmt.Sprintf("0.1.%d", i)
		mt, _ := m.containerImageVersion.GetMetricWith(m.buildLabels("namespace", "pod", "container", typ, "url", version, version))
		count := testutil.ToFloat64(mt)
		if count != 0 {
			t.Error("Should have removed metric")
		}
	}
}
