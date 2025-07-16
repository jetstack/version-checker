package keychains

import (
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNew(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	clientset := fake.NewSimpleClientset()

	tests := []struct {
		name     string
		opts     *ManagerOpts
		expected Manager
	}{
		{
			name: "ManualMode",
			opts: &ManagerOpts{
				Mode:       ManualMode,
				CachingTTL: 5 * time.Minute,
			},
			expected: &ManualKeychain{},
		},
		{
			name: "PodMode",
			opts: &ManagerOpts{
				Mode:       PodMode,
				CachingTTL: 5 * time.Minute,
			},
			expected: &PodKeychain{
				client: clientset,
				log:    log,
				opts:   &ManagerOpts{Mode: PodMode, CachingTTL: 5 * time.Minute},
				cache:  cache.New(5*time.Minute, 10*time.Minute),
			},
		},
		{
			name: "ServiceAccountMode",
			opts: &ManagerOpts{
				Mode:       ServiceAccountMode,
				CachingTTL: 5 * time.Minute,
			},
			expected: &ServiceAccountKeychain{
				client: clientset,
				log:    log,
				opts:   &ManagerOpts{Mode: ServiceAccountMode, CachingTTL: 5 * time.Minute},
				cache:  cache.New(5*time.Minute, 10*time.Minute),
			},
		},
		{
			name: "DefaultMode",
			opts: &ManagerOpts{
				Mode:       0,
				CachingTTL: 5 * time.Minute,
			},
			expected: &ManualKeychain{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := New(log, clientset, tt.opts)

			if tt.expected == nil {
				assert.Nil(t, manager)
			} else {
				assert.IsType(t, tt.expected, manager)
			}
		})
	}
}
