# Configuration

This document describes how to configure version-checker.

## Clients

### OCI

If there are no specific client implementations for the upstream registry
provider, then version-checker will fallback to using a basic OCI client.

You can configure authentication for specific registries by [setting credentials
in Docker's config.json](https://github.com/google/go-containerregistry/blob/c195f151efe3369874c72662cd69ad43ee485128/pkg/authn/README.md#docker-config-auth).

If you're using the Helm chart to deploy version-checker then you can set this
with the `dockerconfigjson` value.

```yaml
dockerconfigjson: |
  {
  	"auths": {
  		"registry.example.com": {
  			"auth": "QXp1cmVEaWFtb25kOmh1bnRlcjI="
  		}
  	}
  }
```
