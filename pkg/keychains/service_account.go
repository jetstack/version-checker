package keychains

import (
	"context"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"

	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type ServiceAccountKeychain struct {
	log    *logrus.Entry
	client kubernetes.Interface

	opts  *ManagerOpts
	cache *cache.Cache
}

// Get retrieves or creates a cached keychain securely.
func (sk *ServiceAccountKeychain) Get(ctx context.Context, _ *corev1.Pod, imageURL string) (authn.Keychain, error) {
	if sk.opts.Mode == ManualMode {
		// If we're running in Manual Mode, then we fall back to the original version-checker
		return nil, nil
	}

	cacheKey := sk.cacheKey(nil)
	if cached, found := sk.cache.Get(cacheKey); found {
		sk.log.WithField("cache", cacheKey).Info("Using cached keychain")
		return cached.(authn.Keychain), nil
	}
	sk.log.Warnf("Creating new keychain for %s\n", cacheKey)

	keychain, err := k8schain.New(ctx, sk.client, k8schain.Options{
		Namespace:          *sk.opts.ServiceAccountNamespace,
		ServiceAccountName: *sk.opts.ServiceAccountName,
		ImagePullSecrets:   *sk.opts.AdditionalImagePullSecrets,
		UseMountSecrets:    sk.opts.UseMountSecrets,
	})
	if err != nil {
		return nil, err
	}

	// Add to the Cache, using the cache instances' defaultExpiration Field
	sk.cache.Set(cacheKey, keychain, cache.DefaultExpiration)

	return keychain, nil
}

// Get a cache key based off the Namespace, ServiceAccountNamer and Image Pull Secrets from the Pod
func (sk *ServiceAccountKeychain) cacheKey(pod *corev1.Pod) string {
	return strings.Join(
		append([]string{*sk.opts.ServiceAccountNamespace, *sk.opts.ServiceAccountName}, *sk.opts.AdditionalImagePullSecrets...),
		"-")
}
