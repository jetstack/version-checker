package acr

import (
	"regexp"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/sirupsen/logrus"
)

var _ api.ImageClientFactory = (*Factory)(nil)
var _ authn.Keychain = (*Factory)(nil)

var (
	HostReg = regexp.MustCompile(`.*\.azurecr\.io|.*\.azurecr\.cn|.*\.azurecr\.de|.*\.azurecr\.us`)
)

type Factory struct {
	opts Options
}

func NewFactory(opts Options) *Factory {
	return &Factory{
		opts: opts,
	}
}

func (f *Factory) Name() string {
	return "acr"
}

func (f *Factory) IsHost(host string) bool {
	return HostReg.MatchString(host)
}

func (f *Factory) NewClient(_ *authn.AuthConfig, log *logrus.Entry) (api.ImageClient, error) {
	return NewClient(f.opts, nil, log)
}

func (f *Factory) Resolve(res authn.Resource) (authn.Authenticator, error) {
	if !f.IsHost(res.RegistryStr()) {
		return nil, nil
	}

	return authn.FromConfig(authn.AuthConfig{
		Username:      f.opts.Username,
		Password:      f.opts.Password,
		RegistryToken: f.opts.RefreshToken,
	}), nil
}
