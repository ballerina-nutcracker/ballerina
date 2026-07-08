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
	"net"
	"strconv"
	"sync"

	"ballerina-lang-go/bir"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
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

	// Default lambda for the optional `name` parameter of attach (defaults to ()).
	runtime.RegisterExternFunction(rt, orgName, moduleName, "$Listener.attach$default$1",
		func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) { return nil, nil })

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.initListener",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			port := int(args[1].(int64))
			config := args[2].(*values.Map)

			state := &listenerState{host: "0.0.0.0", port: port}
			if host := localHostOf(config); host != "" {
				state.host = host
			}
			self.Put("$state", state)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.attach",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			svcObj, ok := args[1].(*values.Object)
			if !ok {
				return udpError("Listener.attach: expected a service object"), nil
			}
			state := listenerStateOf(args[0].(*values.Object))
			state.mu.Lock()
			defer state.mu.Unlock()
			if state.svcObj != nil {
				return udpError("Listener.attach: a service is already attached to this listener"), nil
			}
			state.svcObj = svcObj
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.detach",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			svcObj := args[1].(*values.Object)
			state := listenerStateOf(args[0].(*values.Object))
			state.mu.Lock()
			defer state.mu.Unlock()
			if state.svcObj != svcObj {
				return udpError("Listener.detach: the given service is not attached to this listener"), nil
			}
			state.svcObj = nil
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.start",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			state := listenerStateOf(args[0].(*values.Object))
			state.mu.Lock()
			if state.pc != nil {
				state.mu.Unlock()
				return nil, nil
			}
			if state.svcObj == nil {
				state.mu.Unlock()
				return udpError("Error initializing the server: no service attached"), nil
			}
			svcObj := state.svcObj
			address := fmt.Sprintf("%s:%d", state.host, state.port)
			state.mu.Unlock()

			pc, err := rt.Platform().Net.ListenPacket("udp", address)
			if err != nil {
				return udpError("Unable to initialize UDP Listener: " + err.Error()), nil
			}
			state.mu.Lock()
			state.pc = pc
			state.mu.Unlock()
			go readLoop(rt, types, state, svcObj)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.gracefulStop",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			return stopListener(listenerStateOf(args[0].(*values.Object)), "Failed to gracefully shutdown the Listener."), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.immediateStop",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			return stopListener(listenerStateOf(args[0].(*values.Object)), "Failed to shutdown the Listener."), nil
		})
}

func stopListener(state *listenerState, failureMsg string) values.BalValue {
	state.mu.Lock()
	pc := state.pc
	alreadyStopped := state.stopped
	state.stopped = true
	state.mu.Unlock()
	if alreadyStopped {
		return nil
	}
	if pc == nil {
		return udpError("Unable to initialize the udp listener.")
	}
	if err := pc.Close(); err != nil {
		return udpError(failureMsg)
	}
	return nil
}

func readLoop(rt *runtime.Runtime, types udpTypes, state *listenerState, svcObj *values.Object) {
	for {
		buf := make([]byte, readBufferSize)
		n, addr, err := state.pc.ReadFrom(buf)
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				invokeOnError(rt.NewExternContext(), svcObj, err.Error())
			}
			return
		}
		host, portStr, splitErr := net.SplitHostPort(addr.String())
		if splitErr != nil {
			continue
		}
		port, _ := strconv.ParseInt(portStr, 10, 64)
		data := make([]byte, n)
		copy(data, buf[:n])
		go dispatchDatagram(rt, types, svcObj, state.pc, host, port, data)
	}
}
