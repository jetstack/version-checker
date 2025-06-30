package quay

import (
	"fmt"
	"regexp"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/sirupsen/logrus"
)

var _ api.ImageClientFactory = (*Factory)(nil)
var _ authn.Keychain = (*Factory)(nil)

type Factory struct {
	opts Options
}

var (
	reg = regexp.MustCompile(`(^(.*\.)?quay.io$)`)
)

func NewFactory(opts Options) *Factory {
	return &Factory{opts: opts}
}

func (f *Factory) Name() string {
	return "quay"
}

func (f *Factory) IsHost(host string) bool {
	return reg.MatchString(host)
}

func (f *Factory) NewClient(auth *authn.AuthConfig, log *logrus.Entry) (api.ImageClient, error) {
	// Quay.io Requires authentication for their API - Anonymous is not allowed.
	if (auth == nil || auth == &authn.AuthConfig{}) {
		return nil, fmt.Errorf("client requires authentication, got %T", auth)
	}

	return NewClient(f.opts, auth, log), nil
}

func (f *Factory) Resolve(res authn.Resource) (authn.Authenticator, error) {
	if !f.IsHost(res.RegistryStr()) {
		return nil, nil
	}

	return authn.FromConfig(authn.AuthConfig{
		RegistryToken: f.opts.Token,
	}), nil
}
