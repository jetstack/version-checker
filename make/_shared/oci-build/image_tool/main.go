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
	"os"

	"github.com/spf13/cobra"
)

var CommandRoot = cobra.Command{
	Use: "image-tool",
}

func main() {
	CommandRoot.AddCommand(&CommandAppendLayers)
	CommandRoot.AddCommand(&CommandConvertToDockerTar)
	CommandRoot.AddCommand(&CommandListDigests)
	must("error running command", CommandRoot.Execute())
}

func must(msg string, err error) {
	if err != nil {
		fail(msg+": %w", err)
	}
}

func fail(msg string, a ...any) {
	fmt.Fprintf(os.Stderr, msg+"\n", a...)
	os.Exit(1)
}
