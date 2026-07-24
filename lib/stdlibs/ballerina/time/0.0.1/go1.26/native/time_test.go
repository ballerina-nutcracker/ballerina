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
	"testing"
	"time"

	"ballerina/semtypes"
	"ballerina/values"
)

// The time module's behaviour is exercised end-to-end through the corpus tests
// in corpus/bal/library/subset2/time-*.bal and civil-*.bal, which run the full
// compiler -> BIR -> interpreter pipeline. The unit tests below cover only the
// defensive fallbacks in the argument/value helpers: the Ballerina type checker
// guarantees argument types and arity, so these branches cannot be reached from
// Ballerina source and have no corpus equivalent.

func TestGetInt64ArgFallback(t *testing.T) {
	t.Parallel()
	if got := getInt64Arg(nil, 0); got != 0 {
		t.Errorf("getInt64Arg(nil, 0) = %d, want 0 (index out of range)", got)
	}
	if got := getInt64Arg([]values.BalValue{"not-an-int"}, 0); got != 0 {
		t.Errorf("getInt64Arg(wrong type) = %d, want 0", got)
	}
}

func TestGetStringArgFallback(t *testing.T) {
	t.Parallel()
	if got := getStringArg(nil, 0); got != "" {
		t.Errorf("getStringArg(nil, 0) = %q, want empty (index out of range)", got)
	}
	if got := getStringArg([]values.BalValue{int64(9)}, 0); got != "" {
		t.Errorf("getStringArg(wrong type) = %q, want empty", got)
	}
}

func TestMapIntFallback(t *testing.T) {
	t.Parallel()
	tc := semtypes.ContextFrom(semtypes.CreateTypeEnv())
	m := values.NewMap(semtypes.MAPPING, semtypes.ToMappingAtomicType(tc, semtypes.MAPPING), false, nil)
	if got := mapInt(m, "absent"); got != 0 {
		t.Errorf("mapInt(absent key) = %d, want 0", got)
	}
}

func TestDecimalNilGuards(t *testing.T) {
	t.Parallel()
	if got := decimalToNanos(nil); got != 0 {
		t.Errorf("decimalToNanos(nil) = %d, want 0", got)
	}
	if sec, nano := decimalToSecNano(nil); sec != 0 || nano != 0 {
		t.Errorf("decimalToSecNano(nil) = (%d, %d), want (0, 0)", sec, nano)
	}
}

func TestGetStoredLocationFallback(t *testing.T) {
	t.Parallel()
	// No "$location" field: falls back to UTC.
	empty := values.NewObject(semtypes.OBJECT, nil, nil, nil)
	if loc := getStoredLocation(empty); loc != time.UTC {
		t.Errorf("getStoredLocation(no field) = %v, want UTC", loc)
	}
	// "$location" holding a wrong type: falls back to UTC.
	wrongType := values.NewObject(semtypes.OBJECT, map[string]values.BalValue{"$location": "not-a-location"}, nil, nil)
	if loc := getStoredLocation(wrongType); loc != time.UTC {
		t.Errorf("getStoredLocation(wrong type) = %v, want UTC", loc)
	}
}
