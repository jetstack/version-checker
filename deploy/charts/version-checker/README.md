# version-checker

![Version: v0.5.4](https://img.shields.io/badge/Version-v0.5.4-informational?style=flat-square) ![AppVersion: v0.5.4](https://img.shields.io/badge/AppVersion-v0.5.4-informational?style=flat-square)

A Helm chart for version-checker

**Homepage:** <https://github.com/jetstack/version-checker>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| davidcollom |  |  |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| acr.password | string | `nil` | Password to authenticate with azure container registry |
| acr.refreshToken | string | `nil` | Refresh token to authenticate with azure container registry. Cannot be used with `acr.username` / `acr.password`. |
| acr.username | string | `nil` | Username to authenticate with azure container registry |
| additionalAnnotations | object | `{}` | Additional Annotations to apply to Service and Deployment/Pod Objects |
| additionalLabels | object | `{}` | Additional Labels to apply to Service and Deployment/Pod Objects |
| docker.password | string | `nil` | Password to authenticate with docker registry |
| docker.token | string | `nil` | Token to authenticate with docker registry. Cannot be used with `docker.username` / `docker.password`. |
| docker.username | string | `nil` | Username to authenticate with docker registry |
| ecr.accessKeyID | string | `nil` | ECR access key ID for read access to private registries |
| ecr.iamRoleArn | string | `nil` | Provide AWS EKS Iam Role ARN following: [Specify A ServiceAccount Role](https://docs.aws.amazon.com/eks/latest/userguide/specify-service-account-role.html) |
| ecr.secretAccessKey | string | `nil` | ECR secret access key for read access to private registries |
| ecr.sessionToken | string | `nil` | ECR session token for read access to private registries |
| env | object | `{}` | Can be used to provide custom environment variables e.g. proxy settings |
| gcr.token | string | `nil` | Access token for read access to private GCR registries |
| ghcr.token | string | `nil` | Personal Access token for read access to GHCR releases |
| image.pullPolicy | string | `"IfNotPresent"` | Set the Image Pull Policy |
| image.repository | string | `"quay.io/jetstack/version-checker"` | Repository of the container image |
| image.tag | string | `""` | Override the chart version. Defaults to `appVersion` of the helm chart. |
| livenessProbe.enabled | bool | `true` | Enable/Disable the setting of a livenessProbe |
| livenessProbe.httpGet.path | string | `"/readyz"` | Path to use for the livenessProbe |
| livenessProbe.httpGet.port | int | `8080` | Port to use for the livenessProbe |
| livenessProbe.initialDelaySeconds | int | `3` | Number of seconds after the container has started before liveness probes are initiated. |
| livenessProbe.periodSeconds | int | `3` | How often (in seconds) to perform the livenessProbe. |
| nodeSelector | object | `{}` | Configure nodeSelector |
| podSecurityContext | object | `{}` | Set pod-level security context |
| prometheus.enabled | bool | `false` | Deploy a Prometheus-Operator Prometheus Object to collect version-checker metrics |
| prometheus.replicas | int | `1` | Number of Prometheus replicas to create |
| prometheus.serviceAccountName | string | `"prometheus"` | ServiceAccount for new Prometheus Object |
| quay.token | string | `nil` | Access token for read access to private Quay registries |
| readinessProbe.enabled | bool | `true` | Enable/Disable the setting of a readinessProbe |
| readinessProbe.httpGet.path | string | `"/readyz"` | Path to use for the readinessProbe |
| readinessProbe.httpGet.port | int | `8080` | Port to use for the readinessProbe |
| readinessProbe.initialDelaySeconds | int | `3` | Number of seconds after the container has started before readiness probes are initiated. |
| readinessProbe.periodSeconds | int | `3` | How often (in seconds) to perform the readinessProbe. |
| replicaCount | int | `1` | Replica Count for version-checker |
| resources | object | `{}` | Setup version-checkers resource requests/limits |
| securityContext | object | `{}` | Set container-level security context |
| selfhosted | []{name: "", host: "", username:"", password:"", token:""}] | `[]` | Setup a number of SelfHosted Repositories and their credentials |
| service.annotations | object | `{}` | Additional annotations to add to the service |
| service.labels | object | `{}` | Additional labels to add to the service |
| service.port | int | `8080` | Port to expose within the service |
| serviceMonitor.additionalLabels | object | `{}` | Additional labels to add to the ServiceMonitor |
| serviceMonitor.enabled | bool | `false` | Disable/Enable ServiceMonitor Object |
| tolerations | list | `[]` | Configure tolerations |
| versionChecker.imageCacheTimeout | string | `"30m"` | How long to hold on to image tags and their versions |
| versionChecker.logLevel | string | `"info"` | Configure version-checkers logging, valid options are: debug, info, warn, error, fatal, panic |
| versionChecker.metricsServingAddress | string | `"0.0.0.0:8080"` | Port/interface to which version-checker should bind too |
| versionChecker.testAllContainers | bool | `true` | Enable/Disable the requirement for an enable.version-checker.io annotation on pods. |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.13.1](https://github.com/norwoodj/helm-docs/releases/v1.13.1)
