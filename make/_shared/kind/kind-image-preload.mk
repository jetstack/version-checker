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

ifndef images_amd64
$(error images_amd64 is not set)
endif

ifndef images_arm64
$(error images_arm64 is not set)
endif

##########################################

images := $(images_$(HOST_ARCH))
images_files := $(foreach image,$(images),$(subst :,+,$(image)))

images_tar_dir := $(bin_dir)/downloaded/containers/$(HOST_ARCH)
images_tars := $(images_files:%=$(images_tar_dir)/%.tar)

# Download the images as tarballs. We must use the tag because the digest
# will change after we docker import the image. The tag is the only way to
# reference the image after it has been imported. Before downloading the
# image, we check that the provided digest matches the digest of the image
# that we are about to pull.
$(images_tars): $(images_tar_dir)/%.tar: | $(NEEDS_CRANE)
	@$(eval image=$(subst +,:,$*))
	@$(eval image_without_digest=$(shell cut -d@ -f1 <<<"$(image)"))
	@$(eval digest=$(subst $(image_without_digest)@,,$(image)))
	@mkdir -p $(dir $@)
	diff <(echo "$(digest)  -" | cut -d: -f2) <($(CRANE) manifest --platform=linux/$(HOST_ARCH) $(image_without_digest) | sha256sum)
	$(CRANE) pull $(image_without_digest) $@ --platform=linux/$(HOST_ARCH)

images_tar_envs := $(images_files:%=env-%)

.PHONY: $(images_tar_envs)
$(images_tar_envs): env-%: $(images_tar_dir)/%.tar | $(NEEDS_GOJQ)
	@$(eval image_without_tag=$(shell cut -d+ -f1 <<<"$*"))
	@$(eval $(image_without_tag).TAR="$(images_tar_dir)/$*.tar")
	@$(eval $(image_without_tag).REPO=$(shell tar xfO "$(images_tar_dir)/$*.tar" manifest.json | $(GOJQ) '.[0].RepoTags[0]' -r | cut -d: -f1))
	@$(eval $(image_without_tag).TAG=$(shell tar xfO "$(images_tar_dir)/$*.tar" manifest.json | $(GOJQ) '.[0].RepoTags[0]' -r | cut -d: -f2))
	@$(eval $(image_without_tag).FULL=$($(image_without_tag).REPO):$($(image_without_tag).TAG))

.PHONY: images-preload
## Preload images.
## @category [shared] Kind cluster
images-preload: | $(images_tar_envs)
