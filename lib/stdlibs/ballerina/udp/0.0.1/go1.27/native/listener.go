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
	"net"
	"strconv"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// listenerState is the Go-side state of a udp:Listener object, stored on the
// object's "$state" field. Unlike tcp:Listener, UDP is connectionless: there
// is exactly one socket for the listener's whole lifetime and no per-peer
// connection state to drain, so gracefulStop/immediateStop both reduce to
// closing that one socket.
type listenerState struct {
	host    string
	port    int
	mu      sync.RWMutex
	svcObj  *values.Object
	pc      net.PacketConn
	stopped bool // set by gracefulStop/immediateStop; makes both idempotent
}

func listenerStateOf(self *values.Object) *listenerState {
	v, _ := self.Get("$state")
	state, _ := v.(*listenerState)
	return state
}

func registerListenerFunctions(rt *runtime.Runtime, types udpTypes) {
	listenerClassDef := &bir.BIRClassDef{
		Name:      "Listener",
		LookupKey: "ballerina/udp:Listener",
		VTable: map[string]*bir.BIRFunction{
			"init":          {FunctionLookupKey: "ballerina/udp:Listener.init"},
			"initListener":  {FunctionLookupKey: "ballerina/udp:Listener.initListener"},
			"attach":        {FunctionLookupKey: "ballerina/udp:Listener.attach"},
			"detach":        {FunctionLookupKey: "ballerina/udp:Listener.detach"},
			"start":         {FunctionLookupKey: "ballerina/udp:Listener.start"},
			"gracefulStop":  {FunctionLookupKey: "ballerina/udp:Listener.gracefulStop"},
			"immediateStop": {FunctionLookupKey: "ballerina/udp:Listener.immediateStop"},
		},
	}
	runtime.RegisterExternClassDef(rt, listenerClassDef)
	registerCallerClassDef(rt)

	runtime.RegisterExternFunction(rt, orgName, moduleName, "$Listener.attach$default$1", listenerAttachDefaultName)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.initListener", listenerInit)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.attach", listenerAttach(types))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.detach", listenerDetach)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.start", listenerStart(rt, types))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.gracefulStop", listenerGracefulStop)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.immediateStop", listenerImmediateStop)
}

func listenerAttachDefaultName(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
	return nil, nil
}

func listenerInit(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self := args[0].(*values.Object)
	port := int(args[1].(int64))
	config := args[2].(*values.Map)
	self.Put("$state", &listenerState{host: localHostOf(config), port: port})
	return nil, nil
}

func listenerAttach(types udpTypes) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		svcObj := args[1].(*values.Object)
		if err := validateService(ctx, types, svcObj); err != nil {
			return err, nil
		}
		state := listenerStateOf(args[0].(*values.Object))
		state.mu.Lock()
		defer state.mu.Unlock()
		if state.svcObj != nil {
			return udpError("Listener.attach: a service is already attached to this listener"), nil
		}
		state.svcObj = svcObj
		return nil, nil
	}
}

func listenerDetach(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	svcObj := args[1].(*values.Object)
	state := listenerStateOf(args[0].(*values.Object))
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.svcObj != svcObj {
		return udpError("Listener.detach: the given service is not attached to this listener"), nil
	}
	state.svcObj = nil
	return nil, nil
}

func listenerStart(rt *runtime.Runtime, types udpTypes) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		state := listenerStateOf(args[0].(*values.Object))
		state.mu.Lock()
		defer state.mu.Unlock()
		if state.pc != nil && !state.stopped {
			return nil, nil
		}
		pc, err := rt.Platform().Net.ListenPacket("udp", net.JoinHostPort(state.host, strconv.Itoa(state.port)))
		if err != nil {
			return udpError("Unable to initialize UDP Listener: " + err.Error()), nil
		}
		state.pc = pc
		state.stopped = false
		go readLoop(rt, types, state, pc)
		return nil, nil
	}
}

func listenerGracefulStop(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return stopListener(listenerStateOf(args[0].(*values.Object)), "Failed to gracefully shutdown the Listener."), nil
}

func listenerImmediateStop(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return stopListener(listenerStateOf(args[0].(*values.Object)), "Failed to shutdown the Listener."), nil
}

func stopListener(state *listenerState, failureMsg string) values.BalValue {
	state.mu.Lock()
	pc := state.pc
	if pc == nil || state.stopped {
		state.mu.Unlock()
		return nil
	}
	state.stopped = true
	state.mu.Unlock()
	if err := pc.Close(); err != nil {
		return udpError(failureMsg)
	}
	return nil
}

func readLoop(rt *runtime.Runtime, types udpTypes, state *listenerState, pc net.PacketConn) {
	for {
		buf := make([]byte, readBufferSize)
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) || isStopped(state) {
				return
			}
			if svcObj := attachedService(state); svcObj != nil {
				errCtx := rt.AcquirePooledContext()
				invokeOnError(rt, errCtx, svcObj, err.Error())
				rt.ReleasePooledContext(errCtx)
			}
			continue
		}
		svcObj := attachedService(state)
		if svcObj == nil {
			continue
		}
		data := make([]byte, n)
		copy(data, buf[:n])
		go dispatchDatagram(rt, types, svcObj, pc, addr, data)
	}
}

func attachedService(state *listenerState) *values.Object {
	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.svcObj
}

func isStopped(state *listenerState) bool {
	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.stopped
}
