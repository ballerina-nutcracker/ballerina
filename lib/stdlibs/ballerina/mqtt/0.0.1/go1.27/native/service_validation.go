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
	"fmt"
	"slices"

	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	onMessageMethod  = "onMessage"
	onErrorMethod    = "onError"
	onCompleteMethod = "onComplete"
)

// Messages of the jBallerina compiler plugin diagnostics MQTT_102–MQTT_117.
const (
	msgInvalidRemoteMethod      = "Invalid remote method."
	msgMethodShouldBeRemote     = "Method must have the remote qualifier."
	msgMustHaveMessage          = "Must have the required parameter mqtt:Message and optional parameter mqtt:Caller."
	msgInvalidMessageParam      = "Invalid method parameter. Only mqtt:Message is allowed."
	msgInvalidCallerParam       = "Invalid method parameter. Only mqtt:Caller is allowed"
	msgInvalidMessageParamCount = "Invalid method parameter count. Only mqtt:Caller and mqtt:Message are allowed."
	msgInvalidReturnType        = "Invalid return type. Only error? or mqtt:Error? is allowed."
	msgMustHaveError            = "Must have the required parameter mqtt:Error"
	msgInvalidErrorParamCount   = "Invalid method parameter count. Only mqtt:Error is allowed."
	msgInvalidErrorParam        = "Invalid method parameter. Only mqtt:Error or error is allowed"
	msgInvalidTokenParam        = "Invalid method parameter. Only mqtt:DeliveryToken is allowed"
	msgInvalidTokenParamCount   = "Invalid method parameter count. Only mqtt:DeliveryToken is allowed"
	msgMustHaveToken            = "Must have the required parameter mqtt:DeliveryToken"
)

// serviceParamTypes are the types of the values the dispatcher passes to service methods.
type serviceParamTypes struct {
	message       semtypes.SemType
	caller        semtypes.SemType
	err           semtypes.SemType
	deliveryToken semtypes.SemType
}

// methodSpec lists the parameters a service method may declare, in order; the first
// minParams of them are required.
type methodSpec struct {
	params         []semtypes.SemType
	paramMessages  []string
	minParams      int
	missingMessage string
	countMessage   string
}

// validateService applies the signature rules jBallerina's compiler plugin enforces at
// compile time, so the dispatcher never binds a value to a parameter that cannot hold it.
// A service without onMessage is still accepted and reported through onError at runtime.
func validateService(ctx *extern.Context, svc *values.Object) (*mqttError, error) {
	types, err := lookupServiceParamTypes(ctx)
	if err != nil {
		return nil, err
	}
	specs := map[string]methodSpec{
		onMessageMethod: {
			params:         []semtypes.SemType{types.message, types.caller},
			paramMessages:  []string{msgInvalidMessageParam, msgInvalidCallerParam},
			minParams:      1,
			missingMessage: msgMustHaveMessage,
			countMessage:   msgInvalidMessageParamCount,
		},
		onErrorMethod: {
			params:         []semtypes.SemType{types.err},
			paramMessages:  []string{msgInvalidErrorParam},
			minParams:      1,
			missingMessage: msgMustHaveError,
			countMessage:   msgInvalidErrorParamCount,
		},
		onCompleteMethod: {
			params:         []semtypes.SemType{types.deliveryToken},
			paramMessages:  []string{msgInvalidTokenParam},
			minParams:      1,
			missingMessage: msgMustHaveToken,
			countMessage:   msgInvalidTokenParamCount,
		},
	}
	for _, name := range []string{onMessageMethod, onErrorMethod, onCompleteMethod} {
		if _, ok := svc.MethodLookupKey(name); ok {
			return newError(msgMethodShouldBeRemote), nil
		}
	}
	remoteNames := svc.RemoteMethodNames()
	slices.Sort(remoteNames)
	for _, name := range remoteNames {
		spec, ok := specs[name]
		if !ok {
			return newError(msgInvalidRemoteMethod), nil
		}
		handle, _ := ctx.LookupRemoteMethod(svc, name)
		if mErr := validateMethod(ctx, handle, name, spec); mErr != nil {
			return mErr, nil
		}
	}
	return nil, nil
}

func validateMethod(ctx *extern.Context, handle extern.MethodHandle, name string, spec methodSpec) *mqttError {
	sig, ok := ctx.MethodSignature(handle)
	if !ok {
		return newError(fmt.Sprintf("Unable to resolve the signature of %s", name))
	}
	paramCount := len(sig.Params)
	if sig.RestParam != nil {
		paramCount++
	}
	if paramCount > len(spec.params) {
		return newError(spec.countMessage)
	}
	if paramCount < spec.minParams {
		return newError(spec.missingMessage)
	}
	tc := ctx.TypeCtx()
	for i, p := range sig.Params {
		if !semtypes.IsSubtype(tc, spec.params[i], p.Type) {
			return newError(spec.paramMessages[i])
		}
	}
	if sig.RestParam != nil {
		return newError(spec.paramMessages[len(sig.Params)])
	}
	if !semtypes.IsSubtype(tc, sig.ReturnType, semtypes.Union(semtypes.Error, semtypes.Nil)) {
		return newError(msgInvalidReturnType)
	}
	return nil
}

func lookupServiceParamTypes(ctx *extern.Context) (serviceParamTypes, error) {
	v, err := callHelper(ctx, "serviceParamTypes")
	if err != nil {
		return serviceParamTypes{}, err
	}
	list, ok := v.(*values.List)
	if !ok || list.Len() != 4 {
		return serviceParamTypes{}, fmt.Errorf("mqtt: unexpected service parameter types %v", v)
	}
	constraint := func(i int) semtypes.SemType {
		td, _ := list.Get(i).(*values.TypeDesc)
		return td.Constraint()
	}
	return serviceParamTypes{
		message:       constraint(0),
		caller:        constraint(1),
		err:           constraint(2),
		deliveryToken: constraint(3),
	}, nil
}
