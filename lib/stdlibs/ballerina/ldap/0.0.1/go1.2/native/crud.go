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
	goldap "github.com/go-ldap/ldap/v3"

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

// requireConn returns the live connection for self, or an ldap:Error mirroring
// jBallerina's "LDAP Connection has been closed" check that every remote
// method performs before issuing a request.
func requireConn(self *values.Object) (*goldap.Conn, *values.Error) {
	conn := connOf(self)
	if conn == nil || conn.IsClosing() {
		return nil, ldapError("LDAP Connection has been closed")
	}
	return conn, nil
}

// attributeValueStrings converts a single ldap:AttributeType value
// (boolean|int|float|decimal|string|string[]) to its LDAP wire string form(s),
// reusing this repo's canonical Ballerina value->string conversion so
// formatting matches io:println/string:toString exactly.
func attributeValueStrings(v values.BalValue) []string {
	if list, ok := v.(*values.List); ok {
		vals := make([]string, list.Len())
		for i := 0; i < list.Len(); i++ {
			vals[i] = values.String(list.Get(i), nil)
		}
		return vals
	}
	return []string{values.String(v, nil)}
}

// buildLdapResponse constructs a successful ldap:LdapResponse. go-ldap's
// Add/Del/Modify/ModifyDN calls do not expose the server's raw LDAPResult on
// success (only on failure, via *goldap.Error), so matchedDN/diagnosticMessage/
// referral are left absent — see README "Notable Behavioural Changes".
func buildLdapResponse(types ldapTypes, ctx *extern.Context, operationType string) *values.Map {
	atomic := semtypes.ToMappingAtomicType(ctx.TypeCtx, types.mapTy)
	return values.NewMap(types.mapTy, atomic, false, []values.MapEntry{
		{Key: "matchedDN", Value: nil},
		{Key: "resultCode", Value: "SUCCESS"},
		{Key: "diagnosticMessage", Value: nil},
		{Key: "operationType", Value: operationType},
		{Key: "referral", Value: nil},
	})
}

func registerCrudFunctions(rt *runtime.Runtime, types ldapTypes) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$add",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			dn := args[1].(string)
			entry := args[2].(*values.Map)
			conn, connErr := requireConn(self)
			if connErr != nil {
				return connErr, nil
			}
			addReq := goldap.NewAddRequest(dn, nil)
			for _, key := range entry.Keys() {
				v, _ := entry.Get(key)
				addReq.Attribute(key, attributeValueStrings(v))
			}
			if err := conn.Add(addReq); err != nil {
				return ldapErrorFromErr(err), nil
			}
			return buildLdapResponse(types, ctx, "ADD"), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$delete",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			dn := args[1].(string)
			conn, connErr := requireConn(self)
			if connErr != nil {
				return connErr, nil
			}
			if err := conn.Del(goldap.NewDelRequest(dn, nil)); err != nil {
				return ldapErrorFromErr(err), nil
			}
			return buildLdapResponse(types, ctx, "DELETE"), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$modify",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			dn := args[1].(string)
			entry := args[2].(*values.Map)
			conn, connErr := requireConn(self)
			if connErr != nil {
				return connErr, nil
			}
			modReq := goldap.NewModifyRequest(dn, nil)
			for _, key := range entry.Keys() {
				v, _ := entry.Get(key)
				modReq.Replace(key, attributeValueStrings(v))
			}
			if err := conn.Modify(modReq); err != nil {
				return ldapErrorFromErr(err), nil
			}
			return buildLdapResponse(types, ctx, "MODIFY"), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$modifyDn",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			currentDn := args[1].(string)
			newRdn := args[2].(string)
			deleteOldRdn := false
			if len(args) > 3 {
				if b, ok := args[3].(bool); ok {
					deleteOldRdn = b
				}
			}
			conn, connErr := requireConn(self)
			if connErr != nil {
				return connErr, nil
			}
			modDnReq := goldap.NewModifyDNRequest(currentDn, newRdn, deleteOldRdn, "")
			if err := conn.ModifyDN(modDnReq); err != nil {
				return ldapErrorFromErr(err), nil
			}
			return buildLdapResponse(types, ctx, "MODIFY_DN"), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$compare",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			dn := args[1].(string)
			attrName := args[2].(string)
			assertionValue := args[3].(string)
			conn, connErr := requireConn(self)
			if connErr != nil {
				return connErr, nil
			}
			matched, err := conn.Compare(dn, attrName, assertionValue)
			if err != nil {
				return ldapErrorFromErr(err), nil
			}
			return matched, nil
		})
}
