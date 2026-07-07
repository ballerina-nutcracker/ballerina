// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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
	"ballerina-lang-go/bir"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/semtypes"
)

const (
	orgName    = "ballerina"
	moduleName = "ldap"
)

// ldapTypes holds the SemTypes shared by every native function in this
// module. Following this codebase's established stdlib convention (see
// crypto/file/mime), fixed-shape response records (LdapResponse, SearchResult,
// SearchReference, Control, Entry) are tagged with a single generic open
// mapping type rather than a hand-built per-field SemType: runtime value
// construction here is not re-verified against the declared .bal type on
// assignment, so the extra precision buys nothing at runtime.
type ldapTypes struct {
	mapTy    semtypes.SemType // generic open map, used for LdapResponse/SearchResult/SearchReference/Control/Entry
	strArrTy semtypes.SemType // string[]
	mapArrTy semtypes.SemType // mapTy[], used for entries[]/searchReferences[]/controls[]
}

func initLdapModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	mapBld := semtypes.NewMappingDefinition()
	strArrBld := semtypes.NewListDefinition()
	mapArrBld := semtypes.NewListDefinition()
	mapTy := mapBld.DefineMappingTypeWrapped(env, nil, semtypes.STRING)
	types := ldapTypes{
		mapTy:    mapTy,
		strArrTy: strArrBld.DefineListTypeWrappedWithEnvSemType(env, semtypes.STRING),
		mapArrTy: mapArrBld.DefineListTypeWrappedWithEnvSemType(env, mapTy),
	}

	clientClassDef := &bir.BIRClassDef{
		Name:      "Client",
		LookupKey: "ballerina/ldap:Client",
		VTable: map[string]*bir.BIRFunction{
			"init":                   {FunctionLookupKey: "ballerina/ldap:Client.init"},
			"initLdapConnection":     {FunctionLookupKey: "ballerina/ldap:Client.initLdapConnection"},
			"$remote$add":            {FunctionLookupKey: "ballerina/ldap:Client.$remote$add"},
			"$remote$delete":         {FunctionLookupKey: "ballerina/ldap:Client.$remote$delete"},
			"$remote$modify":         {FunctionLookupKey: "ballerina/ldap:Client.$remote$modify"},
			"$remote$modifyDn":       {FunctionLookupKey: "ballerina/ldap:Client.$remote$modifyDn"},
			"$remote$compare":        {FunctionLookupKey: "ballerina/ldap:Client.$remote$compare"},
			"$remote$getEntry":       {FunctionLookupKey: "ballerina/ldap:Client.$remote$getEntry"},
			"$remote$search":         {FunctionLookupKey: "ballerina/ldap:Client.$remote$search"},
			"$remote$searchWithType": {FunctionLookupKey: "ballerina/ldap:Client.$remote$searchWithType"},
			"$remote$close":          {FunctionLookupKey: "ballerina/ldap:Client.$remote$close"},
			"$remote$isConnected":    {FunctionLookupKey: "ballerina/ldap:Client.$remote$isConnected"},
		},
	}
	runtime.RegisterExternClassDef(rt, clientClassDef)

	registerClientFunctions(rt)
	registerCrudFunctions(rt, types)
	registerSearchFunctions(rt, types)
}

func init() {
	runtime.RegisterModuleInitializer(initLdapModule)
}
