# Metrics

By default, version-checker exposes the following Prometheus metrics on `0.0.0.0:8080/metrics`:

## Container Image Metrics

- `version_checker_is_latest_version`: Indicates whether the container in use is using the latest upstream registry version.
- `version_checker_last_checked`: Timestamp when the image was last checked.
- `version_checker_image_lookup_duration`: Duration of the image version check.
- `version_checker_image_failures_total`: Total of errors encountered during image version checks.
  - Labels: `namespace`, `pod`, `container`, `image`
  - This counter is incremented when version-checker cannot determine the upstream image version, including cases where a registry lookup fails or the image/tag is no longer available upstream.

## Kubernetes Version Metrics

- `version_checker_is_latest_kube_version`: Indicates whether the cluster is running the latest version from the configured Kubernetes release channel.
  - Labels: `current_version`, `latest_version`, `channel`
  - Value `1`: Cluster is up-to-date
  - Value `0`: Update available

---

## Example Prometheus Queries

### Check container image versions
```sh
QUERY="version_checker_is_latest_version"
curl -s --get --data-urlencode query=$QUERY <PROMETHEUS_URL>
```

### Check for failed image lookups
```sh
QUERY='increase(version_checker_image_failures_total[15m]) > 0'
curl -s --get --data-urlencode query="$QUERY" <PROMETHEUS_URL>
```

### Check Kubernetes cluster version
```sh
QUERY="version_checker_is_latest_kube_version"
curl -s --get --data-urlencode query=$QUERY <PROMETHEUS_URL>
```

## Alerting on missing or unavailable images

If a pod references an image tag that has been removed upstream, version-checker will fail the lookup for that image and increment `version_checker_image_failures_total` for the affected `namespace`, `pod`, `container`, and `image`.

Example `PrometheusRule`:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: version-checker-image-failures
spec:
  groups:
    - name: version-checker.rules
      rules:
        - alert: VersionCheckerImageLookupFailures
          expr: increase(version_checker_image_failures_total[15m]) > 0
          for: 15m
          labels:
            severity: warning
          annotations:
            summary: version-checker cannot resolve an upstream image tag
            description: >-
              version-checker has failed to look up the upstream image for
              {{ $labels.namespace }}/{{ $labels.pod }} container
              {{ $labels.container }} (image {{ $labels.image }}) in the last
              15 minutes. This can indicate that the tag has been removed or is
              otherwise unavailable in the registry.
```

To make this alert effective, ensure version-checker is actually checking the containers you care about, either by enabling `--test-all-containers` / `versionChecker.testAllContainers=true` or by opting specific containers in with `version-checker.jetstack.io/enabled`.
