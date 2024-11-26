package metrics

import (
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
)

func TestCache(t *testing.T) {
	m := New(logrus.NewEntry(logrus.New()))

	for i, typ := range []string{"init", "container"} {
		for j := 0; j < 3; j++ {
			version := fmt.Sprintf("0.%d.%d", i, j)
			m.AddImage("namespace", "pod", fmt.Sprintf("%s_container_%d", typ, j), typ, "url", true, version, version)
		}
	}

	for i, typ := range []string{"init", "container"} {
		for j := 0; j < 3; j++ {
			version := fmt.Sprintf("0.%d.%d", i, j)
			mt, _ := m.containerImageVersion.GetMetricWith(m.buildLabels("namespace", "pod", fmt.Sprintf("%s_container_%d", typ, j), typ, "url", version, version))
			count := testutil.ToFloat64(mt)
			if count != 1 {
				t.Error("Should have added metric")
			}
		}
	}

	for i, typ := range []string{"init", "container"} {
		m.RemoveImage("namespace", "pod", fmt.Sprintf("%s_container_0", typ), typ)

		version := fmt.Sprintf("0.%d.0", i)
		mt, _ := m.containerImageVersion.GetMetricWith(m.buildLabels("namespace", "pod", fmt.Sprintf("%s_container_0", typ), typ, "url", version, version))
		count := testutil.ToFloat64(mt)
		if count != 0 {
			t.Error("Should have removed metric")
		}
	}

	for i, typ := range []string{"init", "container"} {
		for j := 1; j < 3; j++ {
			version := fmt.Sprintf("0.%d.%d", i, j)
			mt, _ := m.containerImageVersion.GetMetricWith(m.buildLabels("namespace", "pod", fmt.Sprintf("%s_container_%d", typ, j), typ, "url", version, version))
			count := testutil.ToFloat64(mt)
			if count != 1 {
				t.Error("Should have metric")
			}
		}
	}
}
