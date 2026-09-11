// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package corpus

import (
	"testing"

	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
)

func TestModuleLexicalCompletionCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "completion/testdata/module-lexical.completion.json")
}

func TestCurrentGenerationAfterEditCompletionCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "completion/testdata/current-generation-after-edit.completion.json")
}

func TestExcludedContextsCompletionCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "completion/testdata/excluded-contexts.completion.json")
}

func TestNoResolveOrEditsCompletionCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "completion/testdata/no-resolve-or-edits.completion.json")
}

func TestCancelledWaitRemainsUsableCompletionCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "completion/testdata/cancelled-wait-remains-usable.completion.json")
}

func TestUnusableGenerationCompletionCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "completion/testdata/unusable-generation.completion.json")
}

func TestBlockLexicalCompletionCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "completion/testdata/block-lexical.completion.json")
}
