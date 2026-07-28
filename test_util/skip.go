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

package test_util

import (
	"path/filepath"
	"runtime"
	"strings"
)

// WindowsUnsupportedTests lists corpus tests that cannot run on the Windows CI
// runner but are valid elsewhere — e.g. tests that exec a command which only
// exists on Unix. They are skipped only when GOOS == "windows"; on every other
// platform they run normally. Entries are corpus-relative path suffixes.
var WindowsUnsupportedTests = []string{
	// os:exec runs a real `echo` subprocess, which is not an executable on Windows.
	"library/subset2/os-exec-v.bal",
}

// WASMUnsupportedTests lists corpus tests that cannot run under GOOS=js (the
// WASM CI target) because the WASM sandbox provides no executable binaries.
// Entries are corpus-relative path suffixes.
var WASMUnsupportedTests = []string{
	// os:exec requires spawning a subprocess; no executables exist in the WASM sandbox.
	"library/subset2/os-exec-v.bal",
}

// UnsupportedTests is the single authoritative list of corpus tests that pi
// cannot run end-to-end yet. It is owned by the integration test (which
// exercises the full compile + interpret pipeline) and reused by every
// per-stage corpus test so we don't duplicate skip entries.
//
// Per-stage test files may keep their own *additional* skip list for failures
// that only show up at that stage; they should not re-list entries that are
// already covered here.
//
// Entries are corpus-relative paths (forward slashes, e.g.
// "subset8/08-foo/bar-v.bal"). They are matched as suffixes against either a
// relative test name or an absolute path.
var UnsupportedTests = []string{
	// --- Needs constant folding ---
	// https://github.com/ballerina-nutcracker/ballerina/issues/83

	// pure literal fold + reachability of always-false branch.
	"subset8/08-bitwise/complement3-e.bal",
	"subset8/08-const/1-e.bal",
	"subset8/08-const/7-e.bal",
	"subset8/08-const/8-e.bal",
	"subset8/08-const/9-e.bal",
	"subset8/08-const/10-e.bal",
	"subset8/08-const/11-e.bal",
	"subset8/08-const/12-e.bal",
	"subset8/08-const/13-e.bal",
	"subset8/08-const/14-e.bal",
	"subset8/08-const/15-e.bal",
	"subset8/08-const/16-e.bal",
	"subset8/08-const/17-e.bal",
	"subset8/08-const/18-e.bal",
	"subset8/08-const/7-v.bal",
	"subset8/08-const/8-v.bal",
	"subset8/08-const/10-v.bal",
	"subset8/08-float/5-e.bal",
	"subset8/08-float/7-e.bal",
	"subset8/08-narrowing/unreach3-e.bal",
	"subset8/08-narrowing/unreach4-e.bal",
	"subset8/08-singleton/nil1-e.bal",
	"subset8/08-singleton/stringconcat1-e.bal",
	"subset8/08-string/1-e.bal",
	"subset8/08-string/5-e.bal",

	// singleton narrowing + fold + reachability.
	"subset8/08-narrowing/2-e.bal",
	"subset8/08-narrowing/4-e.bal",
	"subset8/08-narrowing/6-e.bal",
	"subset8/08-narrowing/8-e.bal",
	"subset8/08-narrowing/10-e.bal",
	"subset8/08-narrowing/12-e.bal",
	"subset8/08-narrowing/15-e.bal",
	"subset8/08-singleton/decimal2-e.bal",
	"subset8/08-singleton/decimal4-e.bal",
	"subset8/08-singleton/decimal5-e.bal",
	"subset8/08-singleton/decimal6-e.bal",
	"subset8/08-singleton/decimal7-e.bal",
	"subset8/08-singleton/decimal8-e.bal",
	"subset8/08-singleton/decimal9-e.bal",
	"subset8/08-singleton/decimal10-e.bal",
	"subset8/08-singleton/decimal11-e.bal",
	"subset8/08-singleton/decimal12-e.bal",
	"subset8/08-singleton/decimal13-e.bal",
	"subset8/08-singleton/not1-e.bal",
	"subset8/08-singleton/string1-e.bal",

	// match-arm reachability after discriminator fold/narrowing.
	"subset8/08-match/7-e.bal",
	"subset8/08-match/19-e.bal",

	// disjoint-singleton == / != diagnostic.
	"subset8/08-equal/3-e.bal",
	"subset8/08-equal/4-e.bal",
	"subset8/08-equal/5-e.bal",

	// numeric literal range / typed-cast overflow.
	"subset8/08-const/22-e.bal",
	"subset8/08-const/23-e.bal",
	"subset8/08-decimal/const5-e.bal",
	"subset8/08-decimal/const6-e.bal",
	"subset8/08-hex/decimal1-e.bal",
	"subset8/08-typecast/8-e.bal",

	// const declaration requires singleton-shaped RHS.
	"subset8/08-list/6-e.bal",
	"subset8/08-list/17-e.bal",
	"subset8/08-mapping/6-e.bal",
	"subset8/08-mapping/7-e.bal",

	"subset8/08-decimal/add2-e.bal",
	"subset8/08-decimal/add3-e.bal",
	"subset8/08-decimal/add4-e.bal",
	"subset8/08-decimal/add5-e.bal",
	"subset8/08-decimal/add6-e.bal",
	"subset8/08-decimal/div2-e.bal",
	"subset8/08-decimal/div3-e.bal",
	"subset8/08-decimal/div4-e.bal",
	"subset8/08-decimal/fromfloat2-e.bal",
	"subset8/08-decimal/fromfloat3-e.bal",
	"subset8/08-decimal/mul2-e.bal",
	"subset8/08-decimal/mul3-e.bal",
	"subset8/08-decimal/mul4-e.bal",
	"subset8/08-decimal/mul5-e.bal",
	"subset8/08-decimal/rem3-e.bal",
	"subset8/08-decimal/rem4-e.bal",
	"subset8/08-decimal/sub2-e.bal",
	"subset8/08-decimal/sub3-e.bal",
	"subset8/08-decimal/toint2-e.bal",
	"subset8/08-decimal/toint3-e.bal",
	"subset8/08-decimal/toint4-e.bal",
	"subset8/08-decimal/toint5-e.bal",
	"subset8/08-decimal/toint6-e.bal",

	"subset8/08-const/4-e.bal",
	"subset8/08-const/5-e.bal",
	"subset8/08-const/6-e.bal",
	// ----- End of constant folding -----

	// ----- Float-related skips -----
	// BIR type-tests use plain `float` instead of the singleton union `Special`, so
	// `x is Special` matches every float. Re-enable after TypeTest.Type keeps the union.
	"subset8/08-singleton/floattest1-v.bal",
	"subset8/08-singleton/floattest2-v.bal",

	// ----- End float-related skips -----

	// https://github.com/ballerina-nutcracker/ballerina/issues/283
	"subset8/08-future/fieldexpr1-v.bal",
	// https://github.com/ballerina-nutcracker/ballerina/issues/442
	"subset8/08-future/main-v.bal",
	// https://github.com/ballerina-nutcracker/ballerina/issues/288
	"subset8/08-future/xmlsubtype-v.bal", // xml:Element type unknown

	// Match patterns:
	//  Unsupported match pattern diagnostics for list/mapping patterns.
	// 	https://github.com/ballerina-nutcracker/ballerina/issues/162
	"subset8/08-list/10-e.bal",
	"subset8/08-mapping/9-e.bal",

	// rest param not supported in dependently typed functions
	"subset8/08-function/dependent-fn-5-e.bal",
}

// IsUnsupported reports whether the given corpus test path is in
// UnsupportedTests. The path may be relative (e.g.
// "subset8/08-foo/bar-v.bal") or absolute; entries are matched by suffix.
func IsUnsupported(path string) bool {
	if runtime.GOOS == "windows" && MatchesSkip(path, WindowsUnsupportedTests) {
		return true
	}
	if runtime.GOOS == "js" && MatchesSkip(path, WASMUnsupportedTests) {
		return true
	}
	return MatchesSkip(path, UnsupportedTests)
}

// MatchesSkip reports whether path matches any of the given skip entries by
// suffix. Both path and entries are normalized to forward slashes first.
func MatchesSkip(path string, entries []string) bool {
	p := filepath.ToSlash(path)
	for _, e := range entries {
		if strings.HasSuffix(p, e) {
			return true
		}
	}
	return false
}
