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

package native

import (
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
)

// Not parallel: it lowers the process-wide maximum goroutine stack size.
func TestResolveStopsAtDepthLimit(t *testing.T) {
	const entities = 50000
	var subset strings.Builder
	for i := range entities {
		subset.WriteString("<!ENTITY e" + strconv.Itoa(i) + " \"&e" + strconv.Itoa(i+1) + ";\">")
	}
	subset.WriteString("<!ENTITY e" + strconv.Itoa(entities) + " \"x\">")
	d, message := parseInternalSubset(subset.String())
	if message != "" {
		t.Fatal(message)
	}
	// Resolving the chain without the depth limit would need far more stack than this.
	defer debug.SetMaxStack(debug.SetMaxStack(8 << 20))
	want := "Maximum entity expansion depth limit (500) exceeded"
	if got := d.resolve("e0", map[string]bool{}).message; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
