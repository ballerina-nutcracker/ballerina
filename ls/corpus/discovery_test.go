// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an "AS IS"
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing permissions
// and limitations under the License.

package corpus

import (
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

func TestTranscriptDiscovery(t *testing.T) {
	t.Parallel()
	files := fstest.MapFS{
		"new-feature/testdata/new-sequence.json":     {Data: []byte(`{"messages": [], "expected": []}`)},
		"new-feature/testdata/nested/unrelated.json": {Data: []byte(`not JSON`)},
		"completion/configs/unrelated.json":          {Data: []byte(`not JSON`)},
		"unrelated.json":                             {Data: []byte(`not JSON`)},
	}
	got, err := discoverTranscripts(files)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"new-feature/testdata/new-sequence.json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discovered %v, want %v", got, want)
	}
}

func TestTranscriptDiscoveryRejectsMalformed(t *testing.T) {
	t.Parallel()
	for _, content := range []string{
		`not JSON`, `{}`, `{"messages": []}`, `{"expected": []}`,
		`{"messages": null, "expected": []}`, `{"messages": [], "expected": null}`,
		`{"messages": {}, "expected": []}`, `{"messages": [], "expected": {}}`,
	} {
		t.Run(content, func(t *testing.T) {
			name := "feature/testdata/malformed.json"
			_, err := discoverTranscripts(fstest.MapFS{name: {Data: []byte(content)}})
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("error = %v, want malformed transcript path", err)
			}
		})
	}
}
