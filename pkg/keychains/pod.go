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
	"k8s.io/client-go/rest"
)

type PodKeychain struct {
	log        *logrus.Entry
	restConfig *rest.Config

	client kubernetes.Interface

	opts  *ManagerOpts
	cache *cache.Cache
}

// Get retrieves or creates a cached keychain securely.
func (pk *PodKeychain) Get(ctx context.Context, pod *corev1.Pod, imageURL string) (authn.Keychain, error) {
	cacheKey := pk.cacheKey(pod)

	if cached, found := pk.cache.Get(cacheKey); found {
		pk.log.WithField("cache", cacheKey).Infof("Using cached keychain for %s/%s", pod.Namespace, pod.Name)
		return cached.(authn.Keychain), nil
	}
	pullSecrets := pullSecrets(pod.Spec.ImagePullSecrets)
	pk.log.WithField("pullSecrets", pullSecrets).Warnf("Creating new keychain for %s", cacheKey)

	keychain, err := k8schain.New(ctx, pk.Client(), k8schain.Options{
		Namespace:          pod.Namespace,
		ServiceAccountName: pod.Spec.ServiceAccountName,
		ImagePullSecrets:   pullSecrets,
		UseMountSecrets:    pk.opts.UseMountSecrets,
	})
	if err != nil {
		return nil, err
	}

	// Add to the Cache, using the cache instances' defaultExpiration Field
	pk.cache.Set(cacheKey, keychain, cache.DefaultExpiration)

	return keychain, nil
}

// Get a cache key based off the Namespace, ServiceAccountNamer and Image Pull Secrets from the Pod
func (pk *PodKeychain) cacheKey(pod *corev1.Pod) string {
	return pod.Namespace + "/" + saName(pod.Spec.ServiceAccountName) + "/" + strings.Join(pullSecrets(pod.Spec.ImagePullSecrets), "/")
}

func (pk *PodKeychain) Client() kubernetes.Interface {
	if pk.client != nil {
		return pk.client
	}
	pk.client, _ = kubernetes.NewForConfig(pk.restConfig)
	return pk.client
}
