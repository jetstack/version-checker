# Metrics

By default, version-checker exposes the following Prometheus metrics on `0.0.0.0:8080/metrics`:

- `version_checker_is_latest_version`: Indicates whether the container in use is using the latest upstream registry version.
- `version_checker_last_checked`: Timestamp when the image was last checked.
- `version_checker_image_lookup_duration`: Duration of the image version check.
- `version_checker_image_failures_total`: Total of errors encountered during image version checks.

---

## Example Prometheus Query

```sh
QUERY="version_checker_is_latest_version"
curl -s --get --data-urlencode query=$QUERY <PROMETHEUS_URL>
```