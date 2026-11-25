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

// Package native implements the extern functions of ballerina/auth.
package native

import (
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

const (
	orgName    = "ballerina"
	moduleName = "auth"
)

func initAuthModule(rt *runtime.Runtime) {
	strArrDef := semtypes.NewListDefinition()
	strArrTy := strArrDef.Define(rt.GetTypeEnv(), nil, semtypes.ListRest(semtypes.String))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "initLdapConnection", initLdapConnectionExtern(rt))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "authenticateWithLdap", authenticateWithLdapExtern)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "getLdapGroups", getLdapGroupsExtern(strArrTy))
}

func init() {
	runtime.RegisterModuleInitializer(initAuthModule)
}
