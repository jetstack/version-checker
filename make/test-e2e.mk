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

.PHONY: e2e-setup-docker-registry
e2e-setup-docker-registry: | kind-cluster $(NEEDS_HELM) $(NEEDS_KUBECTL)
	$(HELM) repo add twuni https://helm.twun.io
	$(HELM) upgrade \
		--install \
		--create-namespace \
		-n registry \
		--wait \
		--set service.type=NodePort \
		--set service.nodePort=30443 \
		-f ./make/config/registry/docker-registry-values.yaml \
		docker-registry twuni/docker-registry >/dev/null

.PHONY: install-harbor
e2e-setup-harbor: | kind-cluster $(NEEDS_HELM) $(NEEDS_KUBECTL)
	$(HELM) repo add harbor https://helm.goharbor.io
	$(HELM) upgrade \
		--install \
		--create-namespace \
		-n harbor \
		--wait \
		--set expose.type=nodePort \
		--set expose.tls.enabled=false \
		--set trivy.enabled=false \
		--set registry.credentials.username="user" \
		--set registry.credentials.password="password" \
		--set expose.nodePort.ports.http.nodePort=30443 \
		harbor harbor/harbor >/dev/null

 
INSTALL_OPTIONS += --set image.repository=$(oci_manager_image_name_development)
INSTALL_OPTIONS += -f ./make/config/version-checker-values.yaml

.PHONY: e2e-setup-deps
e2e-setup-deps: | kind-cluster $(NEEDS_KUBECTL)
	$(KUBECTL) apply -f test/e2e/manifests/docker-credentials.yaml
	$(KUBECTL) apply -f test/e2e/manifests/gsa-secret.yaml #TODO replace with local hostPath context
	$(KUBECTL) apply -f test/e2e/manifests/pod-gcs.yaml

is_e2e_test=

# The "install" target can be run on its own with any currently active cluster,
# we can't use any other cluster then a target containing "test-e2e" is run.
# When a "test-e2e*" target is run, the currently active cluster must be the kind
# cluster created by the "kind-cluster" target.
ifeq ($(findstring test-e2e,$(MAKECMDGOALS)),test-e2e)
is_e2e_test = yes
endif


ifdef is_e2e_test
install: kind-cluster oci-load-manager
endif

test-e2e-deps: e2e-setup-docker-registry
test-e2e-deps: e2e-setup-deps
test-e2e-deps: install



.PHONY: test-e2e
## e2e end-to-end tests
## @category Testing
test-e2e: test-e2e-deps | kind-cluster #$(NEEDS_GINKGO) $(NEEDS_KUBECTL)
# $(GINKGO) \
# 	--output-dir=$(ARTIFACTS) \
# 	--focus="$(E2E_FOCUS)" \
# 	--junit-report=junit-go-e2e.xml \
# 	$(EXTRA_GINKGO_FLAGS) \
# 	./test/e2e/ \
# 	-ldflags $(go_manager_ldflags) \
# 	-- \
# 	--istioctl-path $(CURDIR)/$(bin_dir)/scratch/istioctl-$(ISTIO_VERSION) \
# 	--kubeconfig-path $(CURDIR)/$(kind_kubeconfig) \
# 	--kubectl-path $(KUBECTL) \
# 	--runtime-issuance-config-map-name=$(E2E_RUNTIME_CONFIG_MAP_NAME)
