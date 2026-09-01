# Metrics

By default, version-checker exposes the following Prometheus metrics on `0.0.0.0:8080/metrics`:

## Container Image Metrics

- `version_checker_is_latest_version`: Indicates whether the container in use is using the latest upstream registry version.
- `version_checker_last_checked`: Timestamp when the image was last checked.
- `version_checker_image_lookup_duration`: Duration of the image version check.
- `version_checker_image_failures_total`: Total of errors encountered during image version checks.
  - Labels: `namespace`, `pod`, `container`, `image`
  - This counter is incremented when version-checker cannot determine the upstream image version, including cases where a registry lookup fails or the image/tag is no longer available upstream.
- `version_checker_image_timestamp`: Creation timestamp (Unix seconds) of the currently running image.
  - Labels: `namespace`, `pod`, `container`, `container_type`, `image`
  - Only emitted when version-checker can resolve the running image upstream (by digest or tag) **and** the registry reports a valid creation time. Images whose registry does not expose a timestamp are omitted rather than reported as `0`.
  - Combined with `version_checker_is_latest_version`, this surfaces images that are already at the latest version yet were built long ago (potentially abandoned).
- `version_checker_is_available`: Whether the currently running image was found upstream.
  - Labels: `namespace`, `pod`, `container`, `container_type`, `image`
  - Value `1`: The running image (by digest or tag) still exists in the registry.
  - Value `0`: The running image was not found upstream, which can indicate the tag/digest has been deleted and future pulls may fail.
  - Not emitted when availability cannot be determined (e.g. a transient registry lookup error), to avoid false negatives.

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

### Find images no longer available upstream
```sh
QUERY='version_checker_is_available == 0'
curl -s --get --data-urlencode query="$QUERY" <PROMETHEUS_URL>
```

### Find the age (in days) of running images
```sh
QUERY='(time() - version_checker_image_timestamp) / 86400'
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

## Alerting on abandoned images

An image that is already at the latest available version but was built a long time ago may indicate an abandoned or unmaintained image that warrants investigation (for example, a base image that should be rebuilt). Combine `version_checker_image_timestamp` with `version_checker_is_available` to detect these.

Example `PrometheusRule`:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: version-checker-image-age
spec:
  groups:
    - name: version-checker.rules
      rules:
        # The running image no longer exists upstream: future pulls may fail.
        - alert: VersionCheckerImageUnavailable
          expr: version_checker_is_available == 0
          for: 30m
          labels:
            severity: warning
          annotations:
            summary: running image no longer available upstream
            description: >-
              The image for {{ $labels.namespace }}/{{ $labels.pod }} container
              {{ $labels.container }} ({{ $labels.image }}) was not found in its
              registry. The tag or digest may have been deleted, which can cause
              image pull failures on the next restart.

        # Already at latest, but the image is older than 180 days: possibly
        # abandoned upstream.
        - alert: VersionCheckerImagePotentiallyAbandoned
          expr: |
            (time() - version_checker_image_timestamp) > (180 * 24 * 3600)
            and on (namespace, pod, container, image)
              version_checker_is_latest_version == 1
          for: 1h
          labels:
            severity: info
          annotations:
            summary: running image is latest but very old
            description: >-
              The image for {{ $labels.namespace }}/{{ $labels.pod }} container
              {{ $labels.container }} ({{ $labels.image }}) is already the latest
              available version, but was built more than 180 days ago. It may be
              abandoned upstream and worth investigating.
```

> Note: `version_checker_image_timestamp` is only present for images whose registry reports a creation time, so the abandoned-image alert only fires for those. `version_checker_is_available` has broader coverage, as it also resolves images by tag.
