# Copyright 2023 The cert-manager Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

repo_name := github.com/jetstack/version-checker

kind_cluster_name := version-checker
kind_cluster_config := $(bin_dir)/scratch/kind_cluster.yaml

build_names := manager

go_manager_main_dir := ./cmd
go_manager_mod_dir := .
go_manager_ldflags := -X $(repo_name)/internal/version.AppVersion=$(VERSION) -X $(repo_name)/internal/version.GitCommit=$(GITCOMMIT)
oci_manager_base_image_flavor := static
oci_manager_image_name := quay.io/jetstack/version-checker
oci_manager_image_tag := $(VERSION)
oci_manager_image_name_development := version-checker.local/version-checker
oci_platforms := linux/amd64,linux/arm/v7,linux/arm64,linux/ppc64le,linux/s390x

deploy_name := version-checker
deploy_namespace := version-checker

helm_chart_source_dir := deploy/charts/version-checker
helm_chart_name := version-checker
helm_chart_version := $(VERSION)
helm_labels_template_name := version-checker.labels
helm_docs_use_helm_tool := 1
helm_generate_schema := 1
helm_verify_values := 1

golangci_lint_config := .golangci.yaml

define helm_values_mutation_function
$(YQ) \
	'( .image.repository = "$(oci_manager_image_name)" ) | \
	( .image.tag = "$(oci_manager_image_tag)" )' \
	$1 --inplace
endef

images_amd64 ?=
images_arm64 ?=

images_amd64 += docker.io/kong/httpbin:0.1.0@sha256:9d65a5b1955d2466762f53ea50eebae76be9dc7e277217cd8fb9a24b004154f4
images_arm64 += docker.io/kong/httpbin:0.1.0@sha256:c546c8b06c542b615f053b577707cb72ddc875a0731d56d0ffaf840f767322ad

images_amd64 += quay.io/curl/curl:8.5.0@sha256:e40a76dcfa9405678336774130411ca35beba85db426d5755b3cdd7b99d09a7a
images_arm64 += quay.io/curl/curl:8.5.0@sha256:038b0290c9e4a371aed4f9d6993e3548fcfa32b96e9a170bfc73f5da4ec2354d
