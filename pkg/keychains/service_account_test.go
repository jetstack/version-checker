package keychains

import (
	"context"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// "github.com/google/go-containerregistry/pkg/authn/k8schain"

	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes/fake"
)

func TestServiceAccountKeychain_Get_CachedKeychain(t *testing.T) {
	ctx := context.Background()
	mockClient := fake.NewSimpleClientset()
	mockCache := cache.New(cache.NoExpiration, cache.NoExpiration)
	mockLog := logrus.NewEntry(logrus.New())

	opts := &ManagerOpts{
		Mode:                       ServiceAccountMode,
		ServiceAccountNamespace:    ptr("default"),
		ServiceAccountName:         ptr("default"),
		AdditionalImagePullSecrets: &[]string{"secret1", "secret2"},
	}

	keychain := &ServiceAccountKeychain{
		log:    mockLog,
		client: mockClient,
		opts:   opts,
		cache:  mockCache,
	}

	cacheKey := keychain.cacheKey(nil)
	mockCache.Set(cacheKey, authn.NewMultiKeychain(), cache.DefaultExpiration)

	result, err := keychain.Get(ctx, nil, "example.com/image")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected a cached keychain, got nil")
	}
}

func TestServiceAccountKeychain_Get_NewKeychain(t *testing.T) {
	ctx := context.Background()
	mockClient := fake.NewSimpleClientset()
	mockCache := cache.New(cache.NoExpiration, cache.NoExpiration)
	mockLog := logrus.NewEntry(logrus.New())

	opts := &ManagerOpts{
		Mode:                       ServiceAccountMode,
		ServiceAccountNamespace:    ptr("default"),
		ServiceAccountName:         ptr("default"),
		AdditionalImagePullSecrets: &[]string{"secret1", "secret2"},
		UseMountSecrets:            false,
	}

	keychain := &ServiceAccountKeychain{
		log:    mockLog,
		client: mockClient,
		opts:   opts,
		cache:  mockCache,
	}

	result, err := keychain.Get(ctx, nil, "example.com/image")
	require.NoError(t, err)
	require.NotNil(t, result)

	cacheKey := keychain.cacheKey(nil)
	res, found := mockCache.Get(cacheKey)
	assert.True(t, found)
	assert.NotNil(t, res)
}

func ptr(s string) *string {
	return &s
}
