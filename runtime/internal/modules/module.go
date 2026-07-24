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

package modules

import (
	"ballerina/bir"
	"ballerina/runtime/extern"
	"ballerina/semtypes"
	"ballerina/values"
)

type BIRModule struct {
	Pkg     *bir.BIRPackage
	Globals map[string]values.BalValue
}

type ExternFunction struct {
	Name string
	Impl extern.NativeFunc
}

func NewBIRModule(typeCtx semtypes.Context, pkg *bir.BIRPackage) *BIRModule {
	globals := make(map[string]values.BalValue, len(pkg.GlobalVars))
	for key, gv := range pkg.GlobalVars {
		v, ok := values.FillerValue(typeCtx, gv.GetType())
		if ok {
			globals[key] = v
		}
	}
	return &BIRModule{
		Pkg:     pkg,
		Globals: globals,
	}
}
