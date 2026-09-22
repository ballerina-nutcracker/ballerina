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

package bir_test

import (
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// Native dependently typed functions have no local variables because their
// signature is only known at each call site.
func TestPrintFunctionWithoutLocalVars(t *testing.T) {
	for _, flags := range []model.Flag{0, model.FlagAttached} {
		printer := bir.PrettyPrinter{}
		actual := printer.Print(semtypes.ContextFrom(semtypes.CreateTypeEnv()), bir.BIRPackage{
			PackageID: model.DEFAULT,
			Functions: []bir.BIRFunction{{
				Name:         model.Name("baz"),
				OriginalName: model.Name("baz"),
				Flags:        flags,
			}},
		})
		if !strings.Contains(actual, "baz(){") {
			t.Errorf("expected an empty parameter list for flags %v, got:\n%s", flags, actual)
		}
	}
}
