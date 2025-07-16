package ghcr

import (
	"fmt"
	"regexp"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/sirupsen/logrus"
)

// Ensure that our Factory adhere to the ImageClientFactory interface
var _ api.ImageClientFactory = (*Factory)(nil)
var _ authn.Keychain = (*Factory)(nil)

type Factory struct {
	opts Options
}

const (
	HostRegTempl = `^(containers\.[a-zA-Z0-9-]+\.ghe\.com|ghcr\.io)$`
)

var HostReg = regexp.MustCompile(HostRegTempl)

func NewFactory(opts Options) *Factory {
	return &Factory{opts: opts}
}

// Name returns the name of the client, adding suffix if using a custom Hostname.
func (f *Factory) Name() string {
	str := "ghcr"
	if f.opts.Hostname != "" {
		str += ": " + f.opts.Hostname
	}
	return str
}

// IsHost returns true if the client is configured for the given host.
func (f *Factory) IsHost(host string) bool {
	// If we have a custom hostname.
	if f.opts.Hostname != "" && f.opts.Hostname == host {
		return true
	}
	return HostReg.MatchString(host)
}

func (f *Factory) NewClient(auth *authn.AuthConfig, log *logrus.Entry) (api.ImageClient, error) {
	// GHCR Requires authentication for the packages API
	if (auth == nil || auth == &authn.AuthConfig{}) {
		return nil, fmt.Errorf("unable to create a %s Client, client requires authentication, got %T", f.Name(), auth)
	}
	return NewClient(f.opts, auth, log), nil
}

func (f *Factory) Resolve(res authn.Resource) (authn.Authenticator, error) {
	if !f.IsHost(res.RegistryStr()) {
		return nil, nil
	}
	// GHCR supports authentication via a token
	return authn.FromConfig(authn.AuthConfig{
		RegistryToken: f.opts.Token,
	}), nil
}
