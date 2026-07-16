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

package nativepkg

import (
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
)

func initNativepkgModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, "mockorg", "nativepkg", "hello", func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
		return "hello from native Go", nil
	})
}

func init() {
	runtime.RegisterModuleInitializer(initNativepkgModule)
}
