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

# Download the images as tarballs. After downloading the image using
# its digest, we untar the image and modify the .[0].RepoTags[0] value in
# the manifest.json file to have the correct tag (instead of "i-was-a-digest"
# which is set when the image is pulled using its digest). This tag is used
# to reference the image after it has been imported using docker or kind. Otherwise,
# the image would be imported with the tag "i-was-a-digest" which is not very useful.
# We would have to use digests to reference the image everywhere which might
# not always be possible and does not match the default behavior of eg. our helm charts.
# Untarring and modifying manifest.json is a hack and we hope that crane adds an option
# in the future that allows setting the tag on images that are pulled by digest.
# NOTE: the tag is fully determined based on the input, we fully allow the remote
# tag to point to a different digest. This prevents CI from breaking due to upstream
# changes. However, it also means that we can incorrectly combine digests with tags,
# hence caution is advised.
$(images_tars): $(images_tar_dir)/%.tar: | $(NEEDS_IMAGE-TOOL) $(NEEDS_CRANE) $(NEEDS_GOJQ)
	@$(eval full_image=$(subst +,:,$*))
	@$(eval bare_image=$(word 1,$(subst :, ,$(full_image))))
	@$(eval digest=$(word 2,$(subst @, ,$(full_image))))
	@$(eval tag=$(word 2,$(subst :, ,$(word 1,$(subst @, ,$(full_image))))))
	@mkdir -p $(dir $@)
	$(CRANE) pull "$(bare_image)@$(digest)" $@ --platform=linux/$(HOST_ARCH)
	$(IMAGE-TOOL) tag-docker-tar $@ "$(bare_image):$(tag)"

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
