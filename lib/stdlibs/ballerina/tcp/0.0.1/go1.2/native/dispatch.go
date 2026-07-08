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
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"ballerina-lang-go/bir"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

var callerIDCounter atomic.Uint64

func nextCallerID() string {
	return strconv.FormatUint(callerIDCounter.Add(1), 10)
}

// connState is the per-connection state shared between the accept-side read
// loop and the Caller object's close() method. Guards against jBallerina's
// double onClose invocation (fix #2 in the README parity notes): closeOnce
// runs the close+dispatch exactly once regardless of who triggers it first.
type connState struct {
	mu         sync.Mutex
	conn       net.Conn
	closed     bool
	connSvcObj *values.Object // set once onConnect resolves; nil until then
}

func (cs *connState) closeOnce(ctx *extern.Context) {
	cs.mu.Lock()
	if cs.closed {
		cs.mu.Unlock()
		return
	}
	cs.closed = true
	connSvcObj := cs.connSvcObj
	cs.mu.Unlock()
	_ = cs.conn.Close()
	if connSvcObj != nil {
		invokeOnClose(ctx, connSvcObj)
	}
}

func (cs *connState) isClosed() bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.closed
}

// newCallerObject builds a tcp:Caller instance directly, bypassing the
// compiled Ballerina init (see caller.bal — every Caller is constructed by
// native code, never via `new tcp:Caller(...)`). Fields are computed once
// here and reused for the connection's lifetime, fixing jBallerina's
// per-dispatch recomputation, including a potential reverse-DNS lookup on
// every onBytes call (fix #4 in the README parity notes).
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
	obj := values.NewObject(semtypes.OBJECT, fields, methodKeys, nil)
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

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Caller.$remote$writeBytes",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
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
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Caller.$remote$close",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			connStateOf(self).closeOnce(ctx)
			return nil, nil
		})
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

func invokeOnClose(ctx *extern.Context, connSvcObj *values.Object) {
	handle, ok := ctx.LookupRemoteMethod(connSvcObj, "onClose")
	if !ok {
		return
	}
	_, _ = ctx.InvokeMethod(handle, []values.BalValue{connSvcObj})
}

func dispatchOnError(ctx *extern.Context, connSvcObj *values.Object, tcpErr *values.Error) {
	handle, ok := ctx.LookupRemoteMethod(connSvcObj, "onError")
	if !ok {
		return
	}
	_, _ = ctx.InvokeMethod(handle, []values.BalValue{connSvcObj, tcpErr})
}

// dispatchOnBytes invokes the fixed onBytes(Caller, readonly & byte[]) signature.
// jBallerina additionally accepts a bare onBytes(readonly & byte[]) form via
// reflection-driven parameter binding — not supported here, see README.
func dispatchOnBytes(ctx *extern.Context, types tcpTypes, callerObj, connSvcObj *values.Object, cs *connState, data []byte) {
	handle, ok := ctx.LookupRemoteMethod(connSvcObj, "onBytes")
	if !ok {
		return
	}
	dataList := bytesToList(types, ctx, data)
	result, err := ctx.InvokeMethod(handle, []values.BalValue{connSvcObj, callerObj, dataList})
	if err != nil {
		return
	}
	switch v := result.(type) {
	case *values.List:
		if _, werr := cs.conn.Write(listToBytes(v)); werr != nil {
			dispatchOnError(ctx, connSvcObj, tcpError("Failed to send data."))
		}
	case *values.Error:
		// jBallerina logs and continues rather than closing the connection or
		// routing to onError — replicated as-is, see README parity notes.
		fmt.Fprintln(os.Stderr, "tcp: onBytes returned an error:", v.Message)
	}
}

// handleConnection owns one accepted connection end-to-end: onConnect once,
// then a read loop dispatching onBytes/onError, ending in exactly one
// onClose (fix #2) — see connState.closeOnce.
func handleConnection(rt *runtime.Runtime, types tcpTypes, svcObj *values.Object, conn net.Conn) {
	cs := &connState{conn: conn}
	callerObj := newCallerObject(conn, cs)

	ctx := rt.NewExternContext()
	handle, ok := ctx.LookupRemoteMethod(svcObj, "onConnect")
	if !ok {
		_ = conn.Close()
		return
	}
	result, err := ctx.InvokeMethod(handle, []values.BalValue{svcObj, callerObj})
	if err != nil || isBalError(result) {
		// Fix #3: jBallerina leaves the connection open (and reads paused)
		// forever when onConnect errors. Close it instead.
		_ = conn.Close()
		return
	}
	connSvcObj, ok := result.(*values.Object)
	if !ok {
		// onConnect returned () — no ConnectionService requested. The
		// handler is expected to have closed the caller itself if desired.
		return
	}
	cs.mu.Lock()
	cs.connSvcObj = connSvcObj
	cs.mu.Unlock()

	buf := make([]byte, readBufferSize)
	for {
		n, readErr := conn.Read(buf)
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				dispatchOnError(ctx, connSvcObj, tcpError(readErr.Error()))
			}
			cs.closeOnce(ctx)
			return
		}
		chunk := make([]byte, n)
		copy(chunk, buf[:n])
		dispatchOnBytes(ctx, types, callerObj, connSvcObj, cs, chunk)
	}
}
