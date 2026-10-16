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
	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

const (
	orgName    = "ballerina"
	moduleName = "ldap"
)

// ldapTypes holds the inherent types given to values built by this module's externs.
type ldapTypes struct {
	mapTy      semtypes.SemType // map<anydata>, used for LdapResponse/SearchResult/SearchReference/Control
	entryTy    semtypes.SemType // map<AttributeType>, the inherent type of every ldap:Entry
	strArrTy   semtypes.SemType // string[]
	mapArrTy   semtypes.SemType // mapTy[], used for searchReferences[]/controls[]
	entryArrTy semtypes.SemType // entryTy[], used for entries[]
}

func initLdapModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	mapBld := semtypes.NewMappingDefinition()
	entryBld := semtypes.NewMappingDefinition()
	strArrBld := semtypes.NewListDefinition()
	mapArrBld := semtypes.NewListDefinition()
	entryArrBld := semtypes.NewListDefinition()
	mapTy := mapBld.Define(env, nil, semtypes.CreateAnydata(semtypes.ContextFrom(env)))
	strArrTy := strArrBld.Define(env, nil, semtypes.ListRest(semtypes.String))
	entryTy := entryBld.Define(env, nil, attributeType(strArrTy))
	types := ldapTypes{
		mapTy:      mapTy,
		entryTy:    entryTy,
		strArrTy:   strArrTy,
		mapArrTy:   mapArrBld.Define(env, nil, semtypes.ListRest(mapTy)),
		entryArrTy: entryArrBld.Define(env, nil, semtypes.ListRest(entryTy)),
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

// attributeType mirrors ldap:AttributeType (boolean|int|float|decimal|string|string[]).
func attributeType(strArrTy semtypes.SemType) semtypes.SemType {
	ty := semtypes.Union(semtypes.Boolean, semtypes.Int)
	ty = semtypes.Union(ty, semtypes.Float)
	ty = semtypes.Union(ty, semtypes.Decimal)
	ty = semtypes.Union(ty, semtypes.String)
	return semtypes.Union(ty, strArrTy)
}
