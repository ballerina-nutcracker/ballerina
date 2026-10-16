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
	"net"

	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// newCallerObject builds the udp:Caller passed to a handler for one received datagram.
func newCallerObject(pc net.PacketConn, senderAddr net.Addr, remoteHost string, remotePort int64) *values.Object {
	fields := map[string]values.BalValue{
		"remoteHost": remoteHost,
		"remotePort": remotePort,
	}
	methodKeys := map[string]string{
		"$remote$sendBytes":    "ballerina/udp:Caller.$remote$sendBytes",
		"$remote$sendDatagram": "ballerina/udp:Caller.$remote$sendDatagram",
	}
	obj := values.NewObject(semtypes.Object, fields, methodKeys, nil, nil)
	obj.Put("$pc", pc)
	obj.Put("$addr", senderAddr)
	return obj
}

func registerCallerClassDef(rt *runtime.Runtime) {
	callerClassDef := &bir.BIRClassDef{
		Name:      "Caller",
		LookupKey: "ballerina/udp:Caller",
		VTable: map[string]*bir.BIRFunction{
			"$remote$sendBytes":    {FunctionLookupKey: "ballerina/udp:Caller.$remote$sendBytes"},
			"$remote$sendDatagram": {FunctionLookupKey: "ballerina/udp:Caller.$remote$sendDatagram"},
		},
	}
	runtime.RegisterExternClassDef(rt, callerClassDef)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Caller.$remote$sendBytes", callerSendBytes)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Caller.$remote$sendDatagram", callerSendDatagram(rt))
}

func callerSendBytes(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self := args[0].(*values.Object)
	data := listToBytes(args[1].(*values.List))
	pc := callerPacketConnOf(self)
	v, _ := self.Get("$addr")
	if werr := writeFragments(pc, v.(net.Addr), data); werr != nil {
		return udpError("Failed to send data: " + werr.Error()), nil
	}
	return nil, nil
}

func callerSendDatagram(rt *runtime.Runtime) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		self := args[0].(*values.Object)
		host, port, data := datagramFields(args[1].(*values.Map))
		pc := callerPacketConnOf(self)
		addr, err := resolveUDPAddr(rt, host, port)
		if err != nil {
			return udpError("Failed to send data: " + err.Error()), nil
		}
		if werr := writeFragments(pc, addr, data); werr != nil {
			return udpError("Failed to send data: " + werr.Error()), nil
		}
		return nil, nil
	}
}

func callerPacketConnOf(self *values.Object) net.PacketConn {
	v, _ := self.Get("$pc")
	pc, _ := v.(net.PacketConn)
	return pc
}

func logHandlerError(rt *runtime.Runtime, message string) {
	_, _ = rt.Platform().IO.Stderr([]byte("error: " + message + "\n"))
}

func panicMessage(r any) string {
	switch v := r.(type) {
	case *values.Error:
		return v.Message
	case error:
		return v.Error()
	default:
		return fmt.Sprintf("%v", r)
	}
}

// invokeHandler logs a panicking handler instead of letting it crash the process, as jBallerina does.
func invokeHandler(rt *runtime.Runtime, ctx *extern.Context, handle extern.MethodHandle, args []values.BalValue) (result values.BalValue, ok bool) {
	defer func() {
		if rec := recover(); rec != nil {
			logHandlerError(rt, panicMessage(rec))
			result, ok = nil, false
		}
	}()
	result, err := ctx.InvokeMethod(handle, args)
	return result, err == nil
}

func invokeOnError(rt *runtime.Runtime, ctx *extern.Context, svcObj *values.Object, message string) {
	handle, ok := ctx.LookupRemoteMethod(svcObj, "onError")
	if !ok {
		return
	}
	invokeHandler(rt, ctx, handle, []values.BalValue{svcObj, udpError(message)})
}

// applyDispatchResult sends an auto-reply when a handler returns byte[] or a
// Datagram, and logs a returned error, mirroring jBallerina's Dispatcher.handleResult.
func applyDispatchResult(rt *runtime.Runtime, ctx *extern.Context, svcObj *values.Object, pc net.PacketConn, senderAddr net.Addr, result values.BalValue) {
	switch v := result.(type) {
	case *values.List:
		if werr := writeFragments(pc, senderAddr, listToBytes(v)); werr != nil {
			invokeOnError(rt, ctx, svcObj, "Failed to send data.")
		}
	case *values.Map:
		host, port, data := datagramFields(v)
		addr, err := resolveUDPAddr(rt, host, port)
		if err != nil || writeFragments(pc, addr, data) != nil {
			invokeOnError(rt, ctx, svcObj, "Failed to send data.")
		}
	case *values.Error:
		logHandlerError(rt, v.Message)
	}
}

// remoteMethodArgs binds each declared parameter by type, as jBallerina's
// Dispatcher does: an object-typed parameter gets the Caller, anything else
// gets dataValue. This allows the bare and Caller forms in either order.
func remoteMethodArgs(ctx *extern.Context, svcObj *values.Object, methodName string, dataValue values.BalValue, callerObj *values.Object) []values.BalValue {
	methodTy := semtypes.ObjectMemberType(ctx.TypeCtx(), semtypes.StringConst(model.RemoteMethodName(methodName)), svcObj.Type)
	paramListTy := semtypes.FunctionParamListType(ctx.TypeCtx(), methodTy)
	args := []values.BalValue{svcObj}
	for i := 0; ; i++ {
		paramTy := semtypes.ListMemberTypeInnerVal(ctx.TypeCtx(), paramListTy, semtypes.IntConst(int64(i)))
		if semtypes.IsNever(paramTy) {
			break
		}
		if semtypes.IsSubtype(ctx.TypeCtx(), paramTy, semtypes.Object) {
			args = append(args, callerObj)
		} else {
			args = append(args, dataValue)
		}
	}
	return args
}

// dispatchDatagram invokes whichever of onBytes/onDatagram the attached
// service declares for a single received datagram.
func dispatchDatagram(rt *runtime.Runtime, types udpTypes, svcObj *values.Object, pc net.PacketConn, senderAddr net.Addr, data []byte) {
	remoteHost, remotePort, err := hostPortOf(senderAddr)
	if err != nil {
		return
	}
	ctx := rt.AcquirePooledContext()
	defer rt.ReleasePooledContext(ctx)

	if handle, ok := ctx.LookupRemoteMethod(svcObj, "onBytes"); ok {
		callerObj := newCallerObject(pc, senderAddr, remoteHost, remotePort)
		dataList := bytesToList(types, ctx, data, true)
		if result, ok := invokeHandler(rt, ctx, handle, remoteMethodArgs(ctx, svcObj, "onBytes", dataList, callerObj)); ok {
			applyDispatchResult(rt, ctx, svcObj, pc, senderAddr, result)
		}
	}

	if handle, ok := ctx.LookupRemoteMethod(svcObj, "onDatagram"); ok {
		callerObj := newCallerObject(pc, senderAddr, remoteHost, remotePort)
		datagram := newDatagram(types, ctx, remoteHost, remotePort, data, true)
		if result, ok := invokeHandler(rt, ctx, handle, remoteMethodArgs(ctx, svcObj, "onDatagram", datagram, callerObj)); ok {
			applyDispatchResult(rt, ctx, svcObj, pc, senderAddr, result)
		}
	}
}
