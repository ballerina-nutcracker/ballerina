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
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

var callerIDCounter atomic.Uint64

func nextCallerID() string {
	return strconv.FormatUint(callerIDCounter.Add(1), 10)
}

// connState is shared between the read loop and Caller.close so onClose is dispatched exactly once, whichever side closes first.
type connState struct {
	mu         sync.Mutex
	conn       net.Conn
	closed     bool
	connSvcObj *values.Object // set once onConnect resolves; nil until then
}

// closeOnce dispatches onClose before closing the socket, so the peer only observes the close after onClose has run.
func (cs *connState) closeOnce(rt *runtime.Runtime, ctx *extern.Context) {
	cs.mu.Lock()
	if cs.closed {
		cs.mu.Unlock()
		return
	}
	cs.closed = true
	connSvcObj := cs.connSvcObj
	cs.mu.Unlock()
	if connSvcObj != nil {
		invokeRemote(rt, ctx, connSvcObj, "onClose", func() []values.BalValue {
			return []values.BalValue{connSvcObj}
		})
	}
	_ = cs.conn.Close()
}

func (cs *connState) isClosed() bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.closed
}

// newCallerObject builds a tcp:Caller natively; its fields are computed once per connection.
func newCallerObject(conn net.Conn, cs *connState) *values.Object {
	remoteHost, remotePortStr, _ := net.SplitHostPort(conn.RemoteAddr().String())
	localHost, localPortStr, _ := net.SplitHostPort(conn.LocalAddr().String())
	remotePort, _ := strconv.ParseInt(remotePortStr, 10, 64)
	localPort, _ := strconv.ParseInt(localPortStr, 10, 64)
	fields := map[string]values.BalValue{
		"remoteHost": remoteHost,
		"remotePort": remotePort,
		"localHost":  localHost,
		"localPort":  localPort,
		"id":         nextCallerID(),
	}
	methodKeys := map[string]string{
		"$remote$writeBytes": "ballerina/tcp:Caller.$remote$writeBytes",
		"$remote$close":      "ballerina/tcp:Caller.$remote$close",
	}
	obj := values.NewObject(semtypes.Object, fields, methodKeys, nil, nil)
	obj.Put("$connState", cs)
	return obj
}

func registerCallerClassDef(rt *runtime.Runtime) {
	callerClassDef := &bir.BIRClassDef{
		Name:      "Caller",
		LookupKey: "ballerina/tcp:Caller",
		VTable: map[string]*bir.BIRFunction{
			"$remote$writeBytes": {FunctionLookupKey: "ballerina/tcp:Caller.$remote$writeBytes"},
			"$remote$close":      {FunctionLookupKey: "ballerina/tcp:Caller.$remote$close"},
		},
	}
	runtime.RegisterExternClassDef(rt, callerClassDef)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Caller.$remote$writeBytes", callerWriteBytes)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Caller.$remote$close", callerClose(rt))
}

func callerWriteBytes(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self := args[0].(*values.Object)
	data := listToBytes(args[1].(*values.List))
	cs := connStateOf(self)
	if cs.isClosed() {
		return tcpError("Socket connection already closed."), nil
	}
	if _, err := cs.conn.Write(data); err != nil {
		return classifyWriteErr(err), nil
	}
	return nil, nil
}

func callerClose(rt *runtime.Runtime) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		self := args[0].(*values.Object)
		connStateOf(self).closeOnce(rt, ctx)
		return nil, nil
	}
}

func connStateOf(self *values.Object) *connState {
	v, _ := self.Get("$connState")
	cs, _ := v.(*connState)
	return cs
}

func isBalError(v values.BalValue) bool {
	_, ok := v.(*values.Error)
	return ok
}

// invokeRemote calls a service remote method. A Ballerina panic in the method is logged, as jBallerina does, instead of crashing the interpreter.
func invokeRemote(rt *runtime.Runtime, ctx *extern.Context, obj *values.Object, method string, buildArgs func() []values.BalValue) (result values.BalValue, ok bool) {
	handle, found := ctx.LookupRemoteMethod(obj, method)
	if !found {
		return nil, false
	}
	defer func() {
		if rec := recover(); rec != nil {
			logError(rt, panicMessage(rec))
			result, ok = nil, false
		}
	}()
	res, err := ctx.InvokeMethod(handle, buildArgs())
	return res, err == nil
}

func logError(rt *runtime.Runtime, msg string) {
	_, _ = rt.Platform().IO.Stderr([]byte("error: " + msg + "\n"))
}

func panicMessage(rec any) string {
	switch v := rec.(type) {
	case *values.Error:
		return v.Message
	case error:
		return v.Error()
	default:
		return fmt.Sprintf("%v", rec)
	}
}

// withPooledContext runs each dispatch on its own strand context, so a recovered panic never leaves stale state behind for the next dispatch.
func withPooledContext(rt *runtime.Runtime, fn func(ctx *extern.Context)) {
	ctx := rt.AcquirePooledContext()
	defer rt.ReleasePooledContext(ctx)
	fn(ctx)
}

func dispatchOnError(rt *runtime.Runtime, ctx *extern.Context, connSvcObj *values.Object, tcpErr *values.Error) {
	invokeRemote(rt, ctx, connSvcObj, "onError", func() []values.BalValue {
		return []values.BalValue{connSvcObj, tcpErr}
	})
}

// onBytesArgs binds each declared onBytes parameter by type, as jBallerina's Dispatcher.getOnBytesSignature does: an object-typed parameter gets the Caller, anything else gets the data.
func onBytesArgs(ctx *extern.Context, connSvcObj, callerObj *values.Object, dataList *values.List) []values.BalValue {
	methodTy := semtypes.ObjectMemberType(ctx.TypeCtx(), semtypes.StringConst(model.RemoteMethodName("onBytes")), connSvcObj.Type)
	if semtypes.IsZero(methodTy) || !semtypes.IsSubtype(ctx.TypeCtx(), methodTy, semtypes.Function) {
		return []values.BalValue{connSvcObj, callerObj, dataList}
	}
	paramListTy := semtypes.FunctionParamListType(ctx.TypeCtx(), methodTy)
	args := []values.BalValue{connSvcObj}
	for i := 0; ; i++ {
		paramTy := semtypes.ListMemberTypeInnerVal(ctx.TypeCtx(), paramListTy, semtypes.IntConst(int64(i)))
		if semtypes.IsNever(paramTy) {
			break
		}
		if semtypes.IsSubtype(ctx.TypeCtx(), paramTy, semtypes.Object) {
			args = append(args, callerObj)
		} else {
			args = append(args, dataList)
		}
	}
	return args
}

func dispatchOnBytes(rt *runtime.Runtime, ctx *extern.Context, types tcpTypes, callerObj, connSvcObj *values.Object, cs *connState, data []byte) {
	result, ok := invokeRemote(rt, ctx, connSvcObj, "onBytes", func() []values.BalValue {
		return onBytesArgs(ctx, connSvcObj, callerObj, bytesToList(types, ctx, data))
	})
	if !ok {
		return
	}
	switch v := result.(type) {
	case *values.List:
		if _, werr := cs.conn.Write(listToBytes(v)); werr != nil {
			dispatchOnError(rt, ctx, connSvcObj, tcpError("Failed to send data."))
		}
	case *values.Error:
		// jBallerina logs a returned error and keeps the connection open without routing it to onError.
		logError(rt, v.Message)
	}
}

// handleConnection owns one accepted connection: onConnect once, then a read loop dispatching onBytes/onError, ending in exactly one onClose.
func handleConnection(rt *runtime.Runtime, types tcpTypes, svcObj *values.Object, conn net.Conn) {
	cs := &connState{conn: conn}
	callerObj := newCallerObject(conn, cs)

	var result values.BalValue
	var ok bool
	withPooledContext(rt, func(ctx *extern.Context) {
		result, ok = invokeRemote(rt, ctx, svcObj, "onConnect", func() []values.BalValue {
			return []values.BalValue{svcObj, callerObj}
		})
	})
	if !ok || isBalError(result) {
		// jBallerina leaves the connection open with reads paused forever here; close it instead.
		_ = conn.Close()
		return
	}
	connSvcObj, ok := result.(*values.Object)
	if !ok {
		// onConnect returned (): no connection service, so the handler owns closing the caller.
		return
	}
	cs.mu.Lock()
	cs.connSvcObj = connSvcObj
	cs.mu.Unlock()

	buf := make([]byte, readBufferSize)
	for {
		n, readErr := conn.Read(buf)
		if readErr != nil {
			withPooledContext(rt, func(ctx *extern.Context) {
				if !errors.Is(readErr, io.EOF) && !errors.Is(readErr, net.ErrClosed) && !cs.isClosed() {
					dispatchOnError(rt, ctx, connSvcObj, tcpError(readErr.Error()))
				}
				cs.closeOnce(rt, ctx)
			})
			return
		}
		chunk := make([]byte, n)
		copy(chunk, buf[:n])
		withPooledContext(rt, func(ctx *extern.Context) {
			dispatchOnBytes(rt, ctx, types, callerObj, connSvcObj, cs, chunk)
		})
	}
}
