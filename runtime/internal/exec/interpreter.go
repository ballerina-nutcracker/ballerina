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

package exec

import (
	"ballerina/bir"
	"ballerina/runtime/extern"
)

// RunEntrypoints runs the package's init and (if present) main functions on
// a fresh context. A non-nil error? returned by init/main is surfaced as a
// formatted Go error via the panic/recover path so callers see
// "error: <message>" with the call trace intact. This is exec's
// responsibility (and not the runtime's) because the formatting requires
// the call stack to still be on `ctx` when getFormattedError runs.
func RunEntrypoints(pkg bir.BIRPackage, env *extern.Env) (err error) {
	ctx := CreateContext(env)
	cs := ctx.CallStack.(*callStack)
	defer func() {
		if r := recover(); r != nil {
			ctx.ReleaseAllHeldLocks()
			err = getFormattedError(cs, r)
		}
	}()
	if pkg.InitFunction != nil {
		if result := executeFunction(ctx, pkg.InitFunction, nil, nil); result != nil {
			panic(result)
		}
	}
	if pkg.MainFunction != nil {
		if result := executeFunction(ctx, pkg.MainFunction, nil, nil); result != nil {
			panic(result)
		}
	}
	return nil
}
