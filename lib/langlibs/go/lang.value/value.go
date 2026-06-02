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

package value

import (
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.value"
)

func init() {
	runtime.RegisterModuleInitializer(initValueModule)
}

func initValueModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromJsonWithType", fromJsonWithType)
}

func fromJsonWithType(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	td := args[1].(*values.TypeDesc)
	result, convErr := runtime.FromJsonWithType(ctx.TypeCtx, args[0], td.Type)
	if convErr != nil {
		return convErr, nil
	}
	return result, nil
}
