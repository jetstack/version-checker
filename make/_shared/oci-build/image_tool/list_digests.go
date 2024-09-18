/*
Copyright 2023 The cert-manager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/spf13/cobra"
)

var CommandListDigests = cobra.Command{
	Use:   "list-digests oci-path",
	Short: "Outputs the digests for images found inside the tarball",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		ociLayout, err := layout.FromPath(path)
		must("could not load oci directory", err)

		imageIndex, err := ociLayout.ImageIndex()
		must("could not load oci image index", err)

		indexManifest, err := imageIndex.IndexManifest()
		must("could not load oci index manifest", err)

		for _, man := range indexManifest.Manifests {
			fmt.Println(man.Digest)
		}
	},
}
