package gcr

import (
	"regexp"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/sirupsen/logrus"
)

var (
	reg = regexp.MustCompile(`(^(.*\.)?gcr.io$|^(.*\.)?k8s.io$|^(.+)-docker.pkg.dev$)`)
)

// Ensure that our Factory adhere to the ImageClientFactory interface
var _ api.ImageClientFactory = (*Factory)(nil)
var _ authn.Keychain = (*Factory)(nil)

type Factory struct {
	Options
}

func NewFactory(opts Options) *Factory {
	return &Factory{
		Options: opts,
	}
}

func (f *Factory) IsHost(host string) bool {
	return reg.MatchString(host)
}

func (f *Factory) Name() string {
	return "gcr"
}

func (f *Factory) NewClient(auth *authn.AuthConfig, log *logrus.Entry) (api.ImageClient, error) {
	return NewClient(f.Options, auth, log), nil
}

func (f *Factory) Resolve(res authn.Resource) (authn.Authenticator, error) {
	if f.IsHost(res.RegistryStr()) {
		return authn.FromConfig(authn.AuthConfig{IdentityToken: f.Token}), nil
	}
	return nil, nil
}
