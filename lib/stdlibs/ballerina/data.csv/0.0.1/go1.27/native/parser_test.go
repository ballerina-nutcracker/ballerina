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
	goruntime "runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// TestParseShortRowsUnderWideHeaderAllocatesLinearly guards against sizing every row to the header width: rows
// may be shorter than the header, and a 4096-column header over 1000 one-value rows then allocates 220 MB of
// row indexes for 25 KB of input, against 2 MB without them. It is not parallel: it reads process-wide counters.
func TestParseShortRowsUnderWideHeaderAllocatesLinearly(t *testing.T) {
	const columns, rows = 4096, 1000
	var sb strings.Builder
	for i := range columns {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString("c" + strconv.Itoa(i))
	}
	sb.WriteString(strings.Repeat("\n1", rows))
	input := sb.String()

	env := semtypes.CreateTypeEnv()
	types := newCsvTypes(env)
	ty := newTyper(semtypes.ContextFrom(env), types)
	m := &csvModule{types: types}
	root := defineArray(env, defineMap(env, nil, types.anydataTy))

	var before, after goruntime.MemStats
	goruntime.ReadMemStats(&before)
	result, err := m.parseInput(readOptions(nil), ty, root, input, func() string { return input })
	goruntime.ReadMemStats(&after)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.(*values.List).Len(); got != rows {
		t.Fatalf("parsed %d rows, want %d", got, rows)
	}
	if allocated, limit := after.TotalAlloc-before.TotalAlloc, uint64(rows*8<<10); allocated > limit {
		t.Fatalf("parsing %d one-value rows allocated %d bytes, want at most %d", rows, allocated, limit)
	}
}
