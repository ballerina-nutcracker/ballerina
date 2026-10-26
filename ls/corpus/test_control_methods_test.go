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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package corpus

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
)

func TestLegacyTestControlMethodsIgnoredCorpus(t *testing.T) {
	t.Parallel()
	for _, name := range []string{
		"block-request", "flush-requests",
	} {
		t.Run(name, func(t *testing.T) {
			runTranscript(t, "controls/testdata/legacy-"+name+".json")
		})
	}
}

func TestLegacyWriteFileIgnoredCorpus(t *testing.T) {
	t.Parallel()
	runTranscriptWithCheck(t, "controls/testdata/legacy-write-file.json",
		func(_ *workspace.ProjectService, _ *compile.CompilationService, root string) {
			content, err := os.ReadFile(filepath.Join(root, "main.bal"))
			if err != nil {
				t.Fatal(err)
			}
			if len(content) == 0 {
				t.Fatal("legacy writeFile changed the fixture source")
			}
		})
}
