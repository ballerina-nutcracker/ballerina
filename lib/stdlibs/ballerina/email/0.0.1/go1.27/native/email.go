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

package native

import (
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "email"
)

func emailError(msg string) *values.Error {
	return values.NewErrorWithMessage(msg)
}

// emailErrorWithCause keeps the underlying Go error reachable through the Ballerina error's cause.
func emailErrorWithCause(msg string, cause error) *values.Error {
	return values.NewError(semtypes.Error, msg, values.NewErrorWithMessage(cause.Error()), "", nil)
}

func stringField(m *values.Map, key string) (string, bool) {
	v, ok := m.Get(key)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// stringOrList reads a `string|string[]` field as a list; an absent field reads as empty.
func stringOrList(m *values.Map, key string) []string {
	v, ok := m.Get(key)
	if !ok {
		return nil
	}
	switch val := v.(type) {
	case string:
		return []string{val}
	case *values.List:
		out := make([]string, 0, val.Len())
		for i := range val.Len() {
			if s, ok := val.Get(i).(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func initEmailModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "initSmtpClientEndpoint", initSmtpClientExtern(rt, true))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "initNoAuthSmtpClientEndpoint", initSmtpClientExtern(rt, false))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "send", sendExtern(rt))

	strArrDef := semtypes.NewListDefinition()
	strArrTy := strArrDef.Define(rt.GetTypeEnv(), nil, semtypes.ListRest(semtypes.String))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "initPopClientEndpoint", initPopClientExtern(rt))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "popRead", popReadExtern(rt, strArrTy))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "popClose", popCloseExtern)
}

func init() {
	runtime.RegisterModuleInitializer(initEmailModule)
}
