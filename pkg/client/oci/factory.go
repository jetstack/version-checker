package oci

import (
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/sirupsen/logrus"
)

// Ensure that our client adhere to the ImageClientFactory interface
var _ api.ImageClientFactory = (*Factory)(nil)
var _ authn.Keychain = (*Factory)(nil)

type Factory struct {
	opts *Options
}

func NewFactory(opts Options) *Factory {
	return &Factory{opts: &opts}
}

// Name is the name of this client
func (f *Factory) Name() string {
	return "oci"
}

// IsHost always returns true because it supports any host following the OCI Spec
func (f *Factory) IsHost(_ string) bool {
	return true
}

func (f *Factory) NewClient(auth *authn.AuthConfig, log *logrus.Entry) (api.ImageClient, error) {
	return NewClient(f.opts, auth, log)
}

func (f *Factory) Resolve(res authn.Resource) (authn.Authenticator, error) {
	// OCI Doesn't support Auth, right now...
	// We'll likely use the main keychain anyway!
	return nil, nil
}
