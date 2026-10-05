// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package corpus

import (
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
)

func TestSignatureCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	entries, err := platform.FS.ReadDir("signature/testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			t.Run(entry.Name(), func(t *testing.T) { runTranscript(t, platform, "signature/testdata/"+entry.Name()) })
		}
	}
}
