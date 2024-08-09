package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
)

func TestCache(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	m := New(logger)

	type testCase struct {
		name          string
		namespace     string
		pod           string
		container     string
		containerType string
		version       string
		expectCount   float64
		action        func(*Metrics, *Entry)
	}

	tests := []testCase{
		{
			name:          "Add init container metric",
			namespace:     "namespace",
			pod:           "pod",
			container:     "container",
			containerType: "init",
			version:       "0.1.0",
			expectCount:   1,
			action: func(m *Metrics, lbs *Entry) {
				m.AddImage(lbs)
			},
		},
		{
			name:          "Add container metric",
			namespace:     "namespace",
			pod:           "pod",
			container:     "container",
			containerType: "container",
			version:       "0.1.1",
			expectCount:   1,
			action: func(m *Metrics, lbs *Entry) {
				m.AddImage(lbs)
			},
		},
		{
			name:          "Remove init container metric",
			namespace:     "namespace",
			pod:           "pod",
			container:     "container",
			containerType: "init",
			version:       "0.1.0",
			expectCount:   0,
			action: func(m *Metrics, lbs *Entry) {
				m.RemoveImage(lbs.Namespace, lbs.Pod, lbs.Container, lbs.ContainerType)
			},
		},
		{
			name:          "Remove container metric",
			namespace:     "namespace",
			pod:           "pod",
			container:     "container",
			containerType: "container",
			version:       "0.1.1",
			expectCount:   0,
			action: func(m *Metrics, lbs *Entry) {
				m.RemoveImage(lbs.Namespace, lbs.Pod, lbs.Container, lbs.ContainerType)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lbs := &Entry{
				Namespace:      tt.namespace,
				Pod:            tt.pod,
				Container:      tt.container,
				ContainerType:  tt.containerType,
				ImageURL:       "url",
				CurrentVersion: tt.version,
				LatestVersion:  tt.version,
				OS:             "",
				Arch:           "",
				IsLatest:       true,
			}

			// Perform the action
			tt.action(m, lbs)

			// Verify the metric count
			mt, err := m.containerImageVersion.GetMetricWith(m.buildLabels(lbs))
			if err != nil {
				t.Fatalf("Error getting metric: %v", err)
			}
			count := testutil.ToFloat64(mt)
			if count != tt.expectCount {
				t.Errorf("expected metric count %v, got %v", tt.expectCount, count)
			}

			// Clean up by ensuring the metric is removed
			m.RemoveImage(lbs.Namespace, lbs.Pod, lbs.Container, lbs.ContainerType)
			mt, err = m.containerImageVersion.GetMetricWith(m.buildLabels(lbs))
			if err == nil {
				finalCount := testutil.ToFloat64(mt)
				if finalCount != 0 {
					t.Errorf("expected final metric count 0, got %v", finalCount)
				}
			} else if tt.expectCount != 0 {
				t.Errorf("expected to find metric but got error: %v", err)
			}
		})
	}
}
