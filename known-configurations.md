# Image Glossary

This document serves as a reference for annotations required to be applied to various resources for the correct versions to be reported:

## Images / Annotations

### GKE Managed Images (kube-proxy, etc..)

| Annotation | Reason |
|-|-|
| `use-metadata.version-checker.io/kube-proxy="true"` | This is because many GKE images are suffixed as `v.1.26.5-gke.2700` for example. By Default, version-checker removes tags with metadata (`-XXXX`) as these traditionally relate to an Alpha or pre-release image. |

### Grafana: docker.io/grafana/grafana

Grafana has a few images, when version-checker looks up the version, it inadvertently records these tags as latest, enforcing a strict Semantic Version (without a prefix) prevents this.

|Annotation |
|-|
|`match-regex.version-checker.io/grafana='(\d+)\.(\d+)\.(\d+)'` |

### Cert-Manager

This is required as there's an image `608111629` which is detected as the latest, as Semver, this would equate to `608111629.0.0`.

| Annotation |
|-|
| `match-regex.version-checker.io/cert-manager='v(\d+)\.(\d+)\.(\d+)` |

### Kyverno Policy-Reporter

The Policy-Reporter Kyverno has an image `7127761` which is always detected to be the latest. 

| Annotation |
|-|
| `match-regex.version-checker.io/policy-reporter: '(\d+)\.(\d+)\.(\d+)'` |


### Velero

Velero contains an image `1220` which is always the latest. 

| Annotation |
|-|
| `match-regex.version-checker.io/velero: 'v(\d+)\.(\d+)\.(\d+)'` |
