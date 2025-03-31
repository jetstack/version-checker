package keychains

import (
	"context"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
)

// Main interface to what all CredentailManagers should implement.
type Manager interface {
	Get(ctx context.Context, pod *v1.Pod, imageURL string) (authn.Keychain, error)
	cacheKey(pod *corev1.Pod) string
}

type CredentialsMode int

const (
	// Manual Mode is the existing version-checker mode, which takes a global set of values for each client.
	ManualMode CredentialsMode = iota
	// PodMode takes the Identity of the synced pod to discover the registry's credentials.
	PodMode
	// ServiceAccountMode uses a static ServiceAccount's Identity and uses this for ALL Credentials.
	// By Default, this will be the ServiceAccount of which version-checker is running under.
	ServiceAccountMode
)

type Options = ManagerOpts

type ManagerOpts struct {
	Mode       CredentialsMode
	CachingTTL time.Duration

	UseMountSecrets bool

	// ServiceAccountName is the name of the service account to use in ServiceAccountMode.
	// If not set, the service account of the pod will be used.
	ServiceAccountName *string
	// ServiceAccountNamespace is the namespace of the service account to use in ServiceAccountMode.
	// If not set, the namespace of the pod will be used.
	ServiceAccountNamespace *string
	// AdditionalImagePullSecrets is the image pull secrets to use in ServiceAccountMode.
	// If not set, the image pull secrets of the pod will be used.
	AdditionalImagePullSecrets *[]string
}
