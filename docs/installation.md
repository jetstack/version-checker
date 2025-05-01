# Installation

`version-checker` can be installed using either static manifests or Helm.

---

## Install Using Static Manifests

To install using static manifests, run:

```sh
kubectl apply -k ./deploy/yaml
```

## Install Using Helm
To install using Helm, add the Jetstack Helm repository and install the chart:

```sh
helm repo add jetstack https://charts.jetstack.io
"jetstack" has been added to your repositories

helm install version-checker jetstack/version-checker
```

Output:

```sh
NAME: version-checker
LAST DEPLOYED: Wed Jul 12 17:47:41 2023
NAMESPACE: default
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

### Prometheus Operator Integration

The Helm chart supports creating a [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator) `ServiceMonitor` to expose version-checker's metrics.

## Options

By default, version-checker will only test containers where the pod has the annotation
`enable.version-checker.io/*my-container*`, where `*my-container*` is the `name`
of the container in the pod.

However, by passing the following flag,`-a, --test-all-containers` version-checker will test all containers within the cluster.

### Supported Annotations

`version-checker` supports the following annotations to enrich version checking on image tags:

- `pin-major.version-checker.io/my-container: 4`: will pin the major version to
    check to 4 (`v4.0.0`).

- `pin-minor.version-checker.io/my-container: 3`: will pin the minor version to
    check to 3 (`v0.3.0`).

- `pin-patch.version-checker.io/my-container: 23`: will pin the patch version to
    check to 23 (`v0.0.23`).

- `use-metadata.version-checker.io/my-container: "true"`: will allow to search
    for image tags which contain information after the first part of the semver
    string. For example, this can be pre-releases or build metadata
    (`v1.2.4-alpha.0`, `v1.2.3-debian-r3`).

- `use-sha.version-checker.io/my-container: "true"`: will check against the latest
    SHA tag available. Essentially, the latest image by date. This is silently
    set to true if no image tag, or "latest" image tag is set. Cannot be used with
    any other options.

- `match-regex.version-checker.io/my-container: ^v\d+\.\d+\.\d+-debian-`: is
    used for only comparing against image tags which match the regex set. For
    example, the above annotation will only check against image tags which have
    the form of something like `v1.3.4-debian-r30`.
    `use-metadata.version-checker.io` is not required when this is set. All
    other options, apart from URL overrides, are ignored when this is set.

- `override-url.version-checker.io/my-container: docker.io/bitnami/etcd`: is
    used to change the URL for where to lookup where the latest image version
    is. In this example, the current version of `my-container` will be compared
    against the image versions in the `docker.io/bitnami/etcd` registry.

- `resolve-sha-to-tags.version-checker.io/my-container`: is used to
    resolve images specified using sha256 in kubernetes manifests to valid semver
    tags. To enable this the annotation value must be set to "true".
