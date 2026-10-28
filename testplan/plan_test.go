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

package testplan_test

import (
	"reflect"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/testplan"
)

// TestTestPlanTypesStayPlainData catches a field type creeping away from
// string/bool/slice-of-those (e.g. a function value or interface), which
// would silently break any consumer that needs a TestPlan to cross process
// or language boundaries without a serialization step.
func TestTestPlanTypesStayPlainData(t *testing.T) {
	for _, v := range []any{
		testplan.TestPlan{},
		testplan.TestFunction{},
		testplan.SuiteHook{},
		testplan.AfterSuiteHook{},
		testplan.GroupHook{},
	} {
		ty := reflect.TypeOf(v)
		for i := 0; i < ty.NumField(); i++ {
			field := ty.Field(i)
			if !isPlainDataType(field.Type) {
				t.Errorf("%s.%s has type %s, not plain data", ty.Name(), field.Name, field.Type)
			}
		}
	}
}

func isPlainDataType(ty reflect.Type) bool {
	switch ty.Kind() {
	case reflect.String, reflect.Bool:
		return true
	case reflect.Slice:
		return isPlainDataType(ty.Elem())
	case reflect.Struct:
		for i := 0; i < ty.NumField(); i++ {
			if !isPlainDataType(ty.Field(i).Type) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
