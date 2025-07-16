package keychains

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"

	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"
)

func TestPodKeychain_Get_CachedKeychain(t *testing.T) {
	mockCache := cache.New(5*time.Minute, 10*time.Minute)
	mockLog := logrus.NewEntry(logrus.New())
	k8sclient := fake.NewSimpleClientset()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
			Name:      "test-pod",
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "default",
			ImagePullSecrets:   []corev1.LocalObjectReference{{Name: "test-secret"}},
		},
	}

	expectedKeychain := authn.NewMultiKeychain()
	mockCache.Set("test-namespace/default/test-secret", expectedKeychain, cache.DefaultExpiration)

	pk := &PodKeychain{
		log:    mockLog,
		client: k8sclient,
		opts:   &ManagerOpts{},
		cache:  mockCache,
	}

	keychain, err := pk.Get(context.Background(), pod, "test-image")
	require.NoError(t, err)
	assert.Equal(t, expectedKeychain, keychain)
}

func TestPodKeychain_Get_NewKeychain(t *testing.T) {
	mockCache := cache.New(5*time.Minute, 10*time.Minute)
	mockLog := logrus.NewEntry(logrus.New())
	client := fake.NewSimpleClientset()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
			Name:      "test-pod",
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "default",
			ImagePullSecrets:   []corev1.LocalObjectReference{{Name: "test-secret"}},
		},
	}

	pk := &PodKeychain{
		log:    mockLog,
		client: client,
		opts:   &ManagerOpts{},
		cache:  mockCache,
	}

	keychain, err := pk.Get(context.Background(), pod, "test-image")
	assert.NoError(t, err)
	assert.NotNil(t, keychain)

	cacheKey := "test-namespace/default/test-secret"
	cachedKeychain, found := mockCache.Get(cacheKey)
	assert.True(t, found)
	assert.Equal(t, keychain, cachedKeychain)
}

func TestPodKeychain_Get_SecretDeniedCreatingKeychain(t *testing.T) {
	mockCache := cache.New(5*time.Minute, 10*time.Minute)
	mockLog := logrus.NewEntry(logrus.New())

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
			Name:      "test-pod",
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "default",
			ImagePullSecrets:   []corev1.LocalObjectReference{{Name: "test-secret"}},
		},
	}

	resources := []runtime.Object{pod}
	client := fake.NewSimpleClientset(resources...)

	// Simulate an RBAC 403 Forbidden when trying to list pods
	client.PrependReactor("get", "secrets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, apierrors.NewForbidden(
			schema.GroupResource{Group: "", Resource: "secrets"},
			"",
			errors.New("rbac: access denied"),
		)
	})

	pk := &PodKeychain{
		log:    mockLog,
		client: client,
		opts:   &ManagerOpts{},
		cache:  mockCache,
	}

	keychain, err := pk.Get(context.Background(), pod, "test-image")
	assert.Error(t, err)
	assert.Nil(t, keychain)
}
