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

ifndef bin_dir
$(error bin_dir is not set)
endif

ifndef repo_name
$(error repo_name is not set)
endif

ifndef helm_chart_source_dir
$(error helm_chart_source_dir is not set)
endif

ifndef helm_chart_name
$(error helm_chart_name is not set)
endif

ifndef helm_chart_version
$(error helm_chart_version is not set)
endif

ifndef helm_values_mutation_function
$(error helm_values_mutation_function is not set)
endif

##########################################

helm_chart_sources := $(shell find $(helm_chart_source_dir) -maxdepth 1 -type f) $(shell find $(helm_chart_source_dir)/templates -type f)
helm_chart_archive := $(bin_dir)/scratch/image/$(helm_chart_name)-$(helm_chart_version).tgz

$(helm_chart_archive): $(helm_chart_sources) | $(NEEDS_HELM) $(NEEDS_YQ) $(bin_dir)/scratch/image
	$(eval helm_chart_source_dir_versioned := $@.tmp)
	rm -rf $(helm_chart_source_dir_versioned)
	mkdir -p $(dir $(helm_chart_source_dir_versioned))
	cp -a $(helm_chart_source_dir) $(helm_chart_source_dir_versioned)

	$(call helm_values_mutation_function,$(helm_chart_source_dir_versioned)/values.yaml)

	@if ! $(YQ) -oy '.name' $(helm_chart_source_dir_versioned)/Chart.yaml | grep -q '^$(helm_chart_name)$$'; then \
		echo "Chart name does not match the name in the helm_chart_name variable"; \
		exit 1; \
	fi
	
	$(YQ) '.annotations."artifacthub.io/prerelease" = "$(IS_PRERELEASE)"' \
		--inplace $(helm_chart_source_dir_versioned)/Chart.yaml

	mkdir -p $(dir $@)
	$(HELM) package $(helm_chart_source_dir_versioned) \
		--app-version $(helm_chart_version) \
		--version $(helm_chart_version) \
		--destination $(dir $@)

.PHONY: helm-chart
## Create a helm chart
## @category [shared] Helm Chart
helm-chart: $(helm_chart_archive)

ifdef helm_docs_use_helm_tool

helm_tool_header_search ?= ^<!-- AUTO-GENERATED -->
helm_tool_footer_search ?= ^<!-- /AUTO-GENERATED -->

.PHONY: generate-helm-docs
## Generate Helm chart documentation.
## @category [shared] Generate/ Verify
generate-helm-docs: | $(NEEDS_HELM-TOOL)
	$(HELM-TOOL) inject -i $(helm_chart_source_dir)/values.yaml -o $(helm_chart_source_dir)/README.md --header-search "$(helm_tool_header_search)" --footer-search "$(helm_tool_footer_search)"
else
.PHONY: generate-helm-docs
## Generate Helm chart documentation.
## @category [shared] Generate/ Verify
generate-helm-docs: | $(NEEDS_HELM-DOCS)
	$(HELM-DOCS) $(helm_chart_source_dir)/
endif

shared_generate_targets += generate-helm-docs

ifdef helm_generate_schema
.PHONY: generate-helm-schema
## Generate Helm chart schema.
## @category [shared] Generate/ Verify
generate-helm-schema: | $(NEEDS_HELM-TOOL) $(NEEDS_GOJQ)
	$(HELM-TOOL) schema -i $(helm_chart_source_dir)/values.yaml | $(GOJQ) > $(helm_chart_source_dir)/values.schema.json

shared_generate_targets += generate-helm-schema
endif

ifdef helm_verify_values
.PHONY: verify-helm-values
## Verify Helm chart values using helm-tool.
## @category [shared] Generate/ Verify
verify-helm-values: | $(NEEDS_HELM-TOOL) $(NEEDS_GOJQ)
	$(HELM-TOOL) lint -i $(helm_chart_source_dir)/values.yaml -d $(helm_chart_source_dir)/templates -e $(helm_chart_source_dir)/values.linter.exceptions

shared_verify_targets += verify-helm-values
endif

.PHONY: verify-pod-security-standards
## Verify that the Helm chart complies with the pod security standards.
## @category [shared] Generate/ Verify
verify-pod-security-standards: $(helm_chart_archive) | $(NEEDS_KYVERNO) $(NEEDS_KUSTOMIZE) $(NEEDS_HELM)
	$(KYVERNO) apply <($(KUSTOMIZE) build https://github.com/kyverno/policies/pod-security/enforce) \
		--resource <($(HELM) template $(helm_chart_archive)) 2>/dev/null

shared_verify_targets_dirty += verify-pod-security-standards

.PHONY: verify-helm-lint
## Verify that the Helm chart is linted.
## @category [shared] Generate/ Verify
verify-helm-lint: $(helm_chart_archive) | $(NEEDS_HELM)
	$(HELM) lint $(helm_chart_archive)

shared_verify_targets_dirty += verify-helm-lint
