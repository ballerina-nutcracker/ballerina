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

package mathpkg

import (
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
)

func initMathpkgModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, "mockorg", "mathpkg", "add", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		a := args[0].(int64)
		b := args[1].(int64)
		return a + b, nil
	})
	runtime.RegisterExternFunction(rt, "mockorg", "mathpkg", "double", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		n := args[0].(int64)
		return n * 2, nil
	})
}

func init() {
	runtime.RegisterModuleInitializer(initMathpkgModule)
}
