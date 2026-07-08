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
	"os"

	"ballerina-lang-go/bir"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

// newCallerObject builds a udp:Caller instance directly, bypassing the
// compiled Ballerina init (see caller.bal — every Caller is constructed by
// native code, never via `new udp:Caller(...)`). remoteHost/remotePort are
// the sender address of the one datagram this Caller was dispatched for.
func newCallerObject(pc net.PacketConn, remoteHost string, remotePort int64) *values.Object {
	fields := map[string]values.BalValue{
		"remoteHost": remoteHost,
		"remotePort": remotePort,
	}
	methodKeys := map[string]string{
		"$remote$sendBytes":    "ballerina/udp:Caller.$remote$sendBytes",
		"$remote$sendDatagram": "ballerina/udp:Caller.$remote$sendDatagram",
	}
	obj := values.NewObject(semtypes.OBJECT, fields, methodKeys, nil)
	obj.Put("$pc", pc)
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

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Caller.$remote$sendBytes",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			data := listToBytes(args[1].(*values.List))
			pc := callerPacketConnOf(self)
			remoteHost, _ := self.Get("remoteHost")
			remotePort, _ := self.Get("remotePort")
			addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", remoteHost.(string), remotePort.(int64)))
			if err != nil {
				return udpError("Failed to send data: " + err.Error()), nil
			}
			if werr := writeFragments(pc, addr, data); werr != nil {
				return udpError("Failed to send data: " + werr.Error()), nil
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Caller.$remote$sendDatagram",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			host, port, data := datagramFields(args[1].(*values.Map))
			pc := callerPacketConnOf(self)
			addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
			if err != nil {
				return udpError("Failed to send data: " + err.Error()), nil
			}
			if werr := writeFragments(pc, addr, data); werr != nil {
				return udpError("Failed to send data: " + werr.Error()), nil
			}
			return nil, nil
		})
}

func callerPacketConnOf(self *values.Object) net.PacketConn {
	v, _ := self.Get("$pc")
	pc, _ := v.(net.PacketConn)
	return pc
}

func invokeOnError(ctx *extern.Context, svcObj *values.Object, message string) {
	handle, ok := ctx.LookupRemoteMethod(svcObj, "onError")
	if !ok {
		return
	}
	_, _ = ctx.InvokeMethod(handle, []values.BalValue{svcObj, udpError(message)})
}

// applyDispatchResult sends an auto-reply when a handler returns byte[] or a
// Datagram, and logs (without closing anything — there is no per-datagram
// connection to close) when it returns an error, mirroring jBallerina's
// Dispatcher.handleResult.
func applyDispatchResult(ctx *extern.Context, svcObj *values.Object, pc net.PacketConn, senderAddr net.Addr, result values.BalValue) {
	switch v := result.(type) {
	case *values.List:
		if werr := writeFragments(pc, senderAddr, listToBytes(v)); werr != nil {
			invokeOnError(ctx, svcObj, "Failed to send data.")
		}
	case *values.Map:
		host, port, data := datagramFields(v)
		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
		if err != nil || writeFragments(pc, addr, data) != nil {
			invokeOnError(ctx, svcObj, "Failed to send data.")
		}
	case *values.Error:
		fmt.Fprintln(os.Stderr, "udp: handler returned an error:", v.Message)
	}
}

// dispatchDatagram invokes whichever of onBytes/onDatagram the attached
// service declares for a single received datagram — jBallerina dispatches to
// both if both are present. The fixed two-param signature order (data first,
// caller second) matches this module's own README example; jBallerina's
// reflection-driven parameter binding (accepting almost any order/subset) is
// not replicated here.
func dispatchDatagram(rt *runtime.Runtime, types udpTypes, svcObj *values.Object, pc net.PacketConn, remoteHost string, remotePort int64, data []byte) {
	ctx := rt.NewExternContext()
	senderAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", remoteHost, remotePort))
	if err != nil {
		return
	}

	if handle, ok := ctx.LookupRemoteMethod(svcObj, "onBytes"); ok {
		callerObj := newCallerObject(pc, remoteHost, remotePort)
		dataList := bytesToList(types, ctx, data, true)
		result, invokeErr := ctx.InvokeMethod(handle, []values.BalValue{svcObj, dataList, callerObj})
		if invokeErr == nil {
			applyDispatchResult(ctx, svcObj, pc, senderAddr, result)
		}
	}

	if handle, ok := ctx.LookupRemoteMethod(svcObj, "onDatagram"); ok {
		callerObj := newCallerObject(pc, remoteHost, remotePort)
		datagram := newDatagram(types, ctx, remoteHost, remotePort, data, true)
		result, invokeErr := ctx.InvokeMethod(handle, []values.BalValue{svcObj, datagram, callerObj})
		if invokeErr == nil {
			applyDispatchResult(ctx, svcObj, pc, senderAddr, result)
		}
	}
}
