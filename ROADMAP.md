# version-checker roadmap

This document proposes a practical roadmap for `version-checker` based on the current milestones, open issues, labels, and open pull requests.

It is intended to help sequence work, not to act as a release promise.

## Guiding themes

The current `v1` milestone already points in a clear direction:

- stability first
- close high-impact bugs
- improve test coverage and confidence
- make image detection more accurate across architectures and registries
- keep the user experience simple to install and operate

## Near term (H1 / Q3)

### 1. Stabilize current workflows and bug backlog

- Fix OCI and image accuracy issues that affect real-world registries:
  - [#121 Error on OCI style repositories](https://github.com/jetstack/version-checker/issues/121)
  - [#318 MANIFEST_UNKNOWN while using selfhosted registry](https://github.com/jetstack/version-checker/issues/318)
- Fix release and contributor workflow issues:
  - [#406 Helm 4 support for workflows](https://github.com/jetstack/version-checker/issues/406)
- Continue to address long-running stability issues such as:
  - [#76 version-checker seemingly leaks memory and gets oom-killed](https://github.com/jetstack/version-checker/issues/76)

### 2. Land small reviewed improvements that already have passing checks

These are good candidates for the next increment because they are already in flight:

- [#431 Fix selfhosted Helm secret key collisions that broke GitLab PAT auth](https://github.com/jetstack/version-checker/pull/431)
- [#432 Add Helm support for Grafana dashboard annotations](https://github.com/jetstack/version-checker/pull/432)
- [#364 Test e2e](https://github.com/jetstack/version-checker/pull/364), if it can be completed in a small, reviewable slice
- [#387 Implement AuthN and K8S ServiceAccount DockerFile Keychain support](https://github.com/jetstack/version-checker/pull/387), ideally broken into smaller mergeable pieces aligned to [#322](https://github.com/jetstack/version-checker/issues/322)

### 3. Improve architecture-aware image resolution

- Revisit [#60 Add architecture and os checks when fetching tags](https://github.com/jetstack/version-checker/pull/60) in smaller increments
- Detect cluster OS/architecture more reliably
- Prefer tags and manifests that match the running platform
- Reduce false positives where a tag exists but is not usable on the cluster architecture

### 4. Raise the quality bar with better tests

- Make measurable progress on [#190 Increase test coverage](https://github.com/jetstack/version-checker/issues/190)
- Prioritize tests around:
  - registry client behavior
  - OCI manifest/index handling
  - architecture selection
  - Helm chart rendering and secret generation
- Reuse the existing unit, chart, and workflow checks rather than inventing parallel validation paths

## Mid term (Q4)

### 1. Complete OCI-native registry support

- Build on the earlier OCI fallback work from [#159 OCI fallback](https://github.com/jetstack/version-checker/pull/159)
- Move from fallback behavior to first-class OCI handling where practical
- Close the gap on OCI auth, manifest index support, and registry-specific edge cases

### 2. Improve private registry UX

- Progress [#322 Auto discovery of credentials using image pull secrets for private repos](https://github.com/jetstack/version-checker/issues/322)
- Make private registry support easier to enable without duplicating credentials into Helm values for every registry
- Ensure any new auth flows are optional and clearly documented

### 3. Expand registry coverage once core accuracy is solid

- Revisit features that are useful but should not outrank stability work, such as:
  - [#391 Add support for public ecr](https://github.com/jetstack/version-checker/pull/391) after its failing checks are resolved

## Longer term / future-development

- Consider items already sitting outside the current `v1` theme, for example:
  - [#8 Support helm chart version checking](https://github.com/jetstack/version-checker/issues/8)
- Reassess larger feature work only after:
  - OCI support is dependable
  - test coverage has improved
  - the open bug backlog is materially smaller

## Suggested order of execution

1. close high-impact bugs
2. merge small green PRs
3. improve architecture and OCI accuracy
4. increase automated test coverage
5. take on larger auth and registry-expansion work
