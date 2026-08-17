// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	root := flag.String("root", ".", "repository root")
	output := flag.String("output", "compilerpluginregistry/plugins.go", "generated Go output")
	check := flag.Bool("check", false, "fail if the generated file is stale")
	flag.Parse()

	absRoot, err := filepath.Abs(*root)
	if err != nil {
		fatal(err)
	}
	generated, err := generate(absRoot)
	if err != nil {
		fatal(err)
	}
	outputPath := *output
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(absRoot, outputPath)
	}
	if *check {
		existing, err := os.ReadFile(outputPath)
		if err != nil {
			fatal(err)
		}
		if !bytes.Equal(existing, generated) {
			fatal(fmt.Errorf("%s is stale; run go run ./compiler-tools/plugin-gen", outputPath))
		}
		return
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(outputPath, generated, 0o644); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
