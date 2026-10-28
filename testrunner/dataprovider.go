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

package testrunner

import (
	"errors"
	"strconv"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// errDataProviderFailed is a Go-side sentinel only — ported from
// registration_api.bal#registerTestConfig, whose `diagnostics = error("Failed
// to execute the data provider")` is a single fixed message used regardless
// of what actually went wrong (a panic, an error value the provider
// legitimately returned, or an unrecognized shape all collapse to this one
// string, discarding the specific underlying message entirely). Callers
// display the exact, capitalized original text directly as a Result.Message
// literal rather than through this error's own .Error() text, since an
// errors.New argument is linted as a Go-style message (lowercase, no
// trailing punctuation) and would otherwise drift from the real one users
// are meant to see.
var errDataProviderFailed = errors.New("failed to execute the data provider")

// DataProviderFailedMessage is the exact console/report text for a data
// provider invocation failure — see errDataProviderFailed.
const DataProviderFailedMessage = "Failed to execute the data provider"

// dataRow is one (key, args) pair extracted from a data provider's return
// value — confirmed by experiment (not assumed) to come back as either a
// *values.Map (keyed test cases, e.g. `{"one": [1, 1], "two": [2, 4]}`) or a
// *values.List (positional test cases, keyed by stringified index), with
// each entry itself a *values.List of plain Go-native argument values
// (int64, etc. — no wrapper type to unwrap).
type dataRow struct {
	key  string
	args []values.BalValue
}

// dataProviderRows invokes the named data provider function and extracts
// its rows, along with which of the two supported return shapes it used —
// report.bal#getTestType reads the same distinction to decide whether a
// failed row's rerun-JSON sub-key gets single-quoted (see statusreport.go).
// A data provider that panics, legitimately returns an error value (its
// declared return type is itself `error|map<...>|...[][]`), or returns an
// unrecognized shape are all reported identically via errDataProviderFailed
// — mirrors registerTestConfig's own `trap invokeFunction(dataProvider, [])`
// check, which collapses all three into the same `output is error` branch.
func (r *runner) dataProviderRows(name string) (rows []dataRow, testType TestType, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = errDataProviderFailed
		}
	}()

	fn, ok := runtime.LookupFunction(r.rt, r.org, r.module, name)
	if !ok {
		return nil, GeneralTest, errDataProviderFailed
	}
	val, invokeErr := runtime.InvokeFunction(r.rt, fn, nil)
	if invokeErr != nil {
		return nil, GeneralTest, errDataProviderFailed
	}
	if _, isErrVal := val.(*values.Error); isErrVal {
		return nil, GeneralTest, errDataProviderFailed
	}

	switch v := val.(type) {
	case *values.Map:
		testType = DataDrivenMapOfTuple
		for _, k := range v.Keys() {
			entry, _ := v.Get(k)
			rows = append(rows, dataRow{key: k, args: listArgs(entry)})
		}
	case *values.List:
		testType = DataDrivenTupleOfTuple
		for i := 0; i < v.Len(); i++ {
			rows = append(rows, dataRow{key: strconv.Itoa(i), args: listArgs(v.Get(i))})
		}
	default:
		return nil, GeneralTest, errDataProviderFailed
	}
	return rows, testType, nil
}

func listArgs(entry values.BalValue) []values.BalValue {
	list, ok := entry.(*values.List)
	if !ok {
		return nil
	}
	args := make([]values.BalValue, list.Len())
	for i := range args {
		args[i] = list.Get(i)
	}
	return args
}
