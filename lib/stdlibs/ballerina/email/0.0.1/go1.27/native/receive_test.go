// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

package native

import (
	"fmt"
	"strings"
	"testing"
)

func TestSplitEntityUnfoldsLongHeaders(t *testing.T) {
	t.Parallel()
	const lines = 200000
	data := "X-Long: a" + strings.Repeat("\r\n b", lines) + "\r\nSubject: s\r\n\r\nbody"
	headers, body := splitEntity([]byte(data))
	if len(headers) != 2 || headers[0].value != "a"+strings.Repeat(" b", lines) || headers[1].value != "s" {
		t.Fatalf("headers %d, first value length %d", len(headers), len(headers[0].value))
	}
	if string(body) != "body" {
		t.Errorf("body %q", body)
	}
}

func TestParseNodeBoundsNesting(t *testing.T) {
	t.Parallel()
	const levels = 100
	var b strings.Builder
	for i := range levels {
		fmt.Fprintf(&b, "--B%d\r\nContent-Type: multipart/mixed; boundary=B%d\r\n\r\n", i, i+1)
	}
	fmt.Fprintf(&b, "--B%d\r\nContent-Type: text/plain\r\n\r\ninner\r\n--B%d--\r\n", levels, levels)
	for i := levels - 1; i >= 0; i-- {
		fmt.Fprintf(&b, "--B%d--\r\n", i)
	}
	node, err := parseNode([]rawHeader{{"Content-Type", "multipart/mixed; boundary=B0"}}, []byte(b.String()), 0)
	if err != nil {
		t.Fatal(err)
	}
	depth := 0
	for node.multipart {
		if len(node.children) != 1 {
			t.Fatalf("depth %d: %d children, want 1", depth, len(node.children))
		}
		node = node.children[0]
		depth++
	}
	if depth != maxMultipartDepth || node.mediaType != "application/octet-stream" {
		t.Errorf("multipart levels %d ending in %s, want %d ending in application/octet-stream", depth, node.mediaType, maxMultipartDepth)
	}
}
