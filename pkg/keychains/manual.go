package keychains

import (
	"context"

	"github.com/google/go-containerregistry/pkg/authn"
	corev1 "k8s.io/api/core/v1"
)

type ManualKeychain struct{}

func (mk *ManualKeychain) Get(ctx context.Context, pod *corev1.Pod, imageURL string) (authn.Keychain, error) {
	return nil, nil
}

func (mk *ManualKeychain) cacheKey(_ *corev1.Pod) string {
	return ""
}
