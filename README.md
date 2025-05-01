# version-checker

![GitHub Release](https://img.shields.io/github/v/release/jetstack/version-checker)
[![Go Report Card](https://goreportcard.com/badge/github.com/jetstack/version-checker)](https://goreportcard.com/report/github.com/jetstack/version-checker)
[![Tests](https://github.com/jetstack/version-checker/actions/workflows/build-test.yaml/badge.svg)](https://github.com/jetstack/version-checker/actions/workflows/build-test.yaml?query=branch%3Amain)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/jetstack/version-checker)

version-checker is a Kubernetes utility for observing the current versions of
images running in the cluster, as well as the latest available upstream. These
checks get exposed as Prometheus metrics to be viewed on a dashboard, or _soft_
alert cluster operators.

---

## Why Use version-checker?

- **Improved Security**: Ensures images are up-to-date, reducing the risk of using vulnerable or compromised versions.
- **Enhanced Visibility**: Provides a clear overview of all running container versions across clusters.
- **Operational Efficiency**: Automates image tracking and reduces manual intervention in version management.
- **Compliance and Policy**: Enforcement: Helps maintain version consistency and adherence to organizational policies.
- **Incremental Upgrades**: Facilitates frequent, incremental updates to reduce the risk of large, disruptive upgrades.
- **Add-On Compatibility**: Ensures compatibility with the latest versions of Kubernetes add-ons and dependencies.

---

## Registries Supported

version-checker supports the following registries:

- [ACR](https://azure.microsoft.com/en-us/services/container-registry/)
- [Docker Hub](https://hub.docker.com/)
- [ECR](https://aws.amazon.com/ecr/)
- [GCR](https://cloud.google.com/container-registry/) (inc gcr facades such as k8s.gcr.io)
- [Quay](https://quay.io/)
- Self Hosted (Docker V2 API compliant registries, e.g.
  [registry](https://hub.docker.com/_/registry),
  [artifactory](https://jfrog.com/artifactory/) etc.). Multiple self hosted
  registries can be configured at once.

These registries support authentication.

---

## Documentation

- [Installation Guide](docs/installation.md)
- [Metrics](docs/metrics.md)

---

### Grafana Dashboard

A [grafana dashboard](https://grafana.com/grafana/dashboards/12833) is also
available to view the image versions as a table.

![](img/grafana.jpg)
<center></center>
<p align="center">
  <b>Grafana Dashboard</b><br>
</p>

## Known configurations

From time to time, version-checker may need some of the above options applied to determine the latest version,
depending on how the maintainers publish their images. We are making a conscious effort to collate some of these configurations.

See [known-configurations.md](../known-configurations.md) for more details.
