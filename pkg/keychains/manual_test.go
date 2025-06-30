package keychains

import (
	"context"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestManualKeychain_Get(t *testing.T) {
	mk := &ManualKeychain{}

	tests := []struct {
		name      string
		pod       *corev1.Pod
		imageURL  string
		wantKey   authn.Keychain
		wantError bool
	}{
		{
			name:      "nil pod and empty imageURL",
			pod:       nil,
			imageURL:  "",
			wantKey:   nil,
			wantError: false,
		},
		{
			name:      "non-nil pod and valid imageURL",
			pod:       &corev1.Pod{},
			imageURL:  "example.com/image",
			wantKey:   nil,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKey, err := mk.Get(context.Background(), tt.pod, tt.imageURL)
			if (err != nil) != tt.wantError {
				t.Errorf("ManualKeychain.Get() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if gotKey != tt.wantKey {
				t.Errorf("ManualKeychain.Get() = %v, want %v", gotKey, tt.wantKey)
			}
		})
	}
}
func TestManualKeychain_cacheKey(t *testing.T) {
	mk := &ManualKeychain{}

	tests := []struct {
		name string
		pod  *corev1.Pod
		want string
	}{
		{
			name: "nil pod",
			pod:  nil,
			want: "",
		},
		{
			name: "empty pod",
			pod:  &corev1.Pod{},
			want: "",
		},
		{
			name: "pod with name and namespace",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-namespace",
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mk.cacheKey(tt.pod); got != tt.want {
				t.Errorf("ManualKeychain.cacheKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
