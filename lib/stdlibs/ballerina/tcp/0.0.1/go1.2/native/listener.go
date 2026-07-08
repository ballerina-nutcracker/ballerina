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
	"sync"
	"time"

	"ballerina-lang-go/bir"
	"ballerina-lang-go/platform/pal"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
)

// gracefulStopDrainTimeout bounds how long gracefulStop waits for
// in-flight connections to close on their own before returning anyway.
const gracefulStopDrainTimeout = 30 * time.Second

// listenerState is the Go-side state of a tcp:Listener object, stored on the
// object's "$state" field — mirrors http:Listener's listenerState
// (lib/stdlibs/ballerina/http/0.0.1/go1.2/native/http_server.go). Unlike
// http, tcp has no path-based routing: only one service may be attached.
type listenerState struct {
	host    string
	port    int
	tlsCfg  *pal.ServerTLSConfig
	mu      sync.RWMutex
	svcObj  *values.Object
	ln      net.Listener
	conns   map[net.Conn]struct{}
	stopped bool // set by gracefulStop/immediateStop; makes both idempotent
}

func stateOf(self *values.Object) *listenerState {
	v, _ := self.Get("$state")
	state, _ := v.(*listenerState)
	return state
}

func registerListenerFunctions(rt *runtime.Runtime, types tcpTypes) {
	listenerClassDef := &bir.BIRClassDef{
		Name:      "Listener",
		LookupKey: "ballerina/tcp:Listener",
		VTable: map[string]*bir.BIRFunction{
			"init":          {FunctionLookupKey: "ballerina/tcp:Listener.init"},
			"initListener":  {FunctionLookupKey: "ballerina/tcp:Listener.initListener"},
			"attach":        {FunctionLookupKey: "ballerina/tcp:Listener.attach"},
			"detach":        {FunctionLookupKey: "ballerina/tcp:Listener.detach"},
			"start":         {FunctionLookupKey: "ballerina/tcp:Listener.start"},
			"gracefulStop":  {FunctionLookupKey: "ballerina/tcp:Listener.gracefulStop"},
			"immediateStop": {FunctionLookupKey: "ballerina/tcp:Listener.immediateStop"},
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

			state := &listenerState{host: "0.0.0.0", port: port, conns: map[net.Conn]struct{}{}}
			if v, ok := config.Get("localHost"); ok {
				if s, ok := v.(string); ok && s != "" {
					state.host = s
				}
			}
			if v, ok := config.Get("secureSocket"); ok {
				if ssMap, ok := v.(*values.Map); ok {
					tlsCfg, tlsErr := buildListenerTLSConfig(rt, ssMap)
					if tlsErr != nil {
						return tlsErr, nil
					}
					state.tlsCfg = tlsCfg
				}
			}
			self.Put("$state", state)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.attach",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			svcObj, ok := args[1].(*values.Object)
			if !ok {
				return tcpError("Listener.attach: expected a service object"), nil
			}
			state := stateOf(args[0].(*values.Object))
			state.mu.Lock()
			defer state.mu.Unlock()
			if state.svcObj != nil {
				return tcpError("Listener.attach: a service is already attached to this listener"), nil
			}
			state.svcObj = svcObj
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.detach",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			svcObj := args[1].(*values.Object)
			state := stateOf(args[0].(*values.Object))
			state.mu.Lock()
			defer state.mu.Unlock()
			if state.svcObj != svcObj {
				return tcpError("Listener.detach: the given service is not attached to this listener"), nil
			}
			state.svcObj = nil
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.start",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			state := stateOf(args[0].(*values.Object))
			state.mu.Lock()
			if state.ln != nil {
				state.mu.Unlock()
				return nil, nil
			}
			if state.svcObj == nil {
				state.mu.Unlock()
				return tcpError("Error initializing the server: no service attached"), nil
			}
			svcObj := state.svcObj
			address := fmt.Sprintf("%s:%d", state.host, state.port)
			tlsCfg := state.tlsCfg
			state.mu.Unlock()

			ln, err := rt.Platform().Net.Listen("tcp", address, tlsCfg)
			if err != nil {
				return tcpError("Error initializing the server: " + err.Error()), nil
			}
			state.mu.Lock()
			state.ln = ln
			state.mu.Unlock()
			go acceptLoop(rt, types, state, svcObj)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.gracefulStop",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			state := stateOf(args[0].(*values.Object))
			state.mu.Lock()
			ln := state.ln
			alreadyStopped := state.stopped
			state.stopped = true
			state.mu.Unlock()
			if alreadyStopped {
				return nil, nil
			}
			if ln == nil {
				return tcpError("Unable to initialize the tcp listener."), nil
			}
			if err := ln.Close(); err != nil {
				return tcpError("Failed to gracefully shutdown the Listener."), nil
			}
			deadline := time.Now().Add(gracefulStopDrainTimeout)
			for time.Now().Before(deadline) {
				state.mu.RLock()
				remaining := len(state.conns)
				state.mu.RUnlock()
				if remaining == 0 {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.immediateStop",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			state := stateOf(args[0].(*values.Object))
			state.mu.Lock()
			ln := state.ln
			alreadyStopped := state.stopped
			state.stopped = true
			conns := make([]net.Conn, 0, len(state.conns))
			for c := range state.conns {
				conns = append(conns, c)
			}
			state.mu.Unlock()
			if alreadyStopped {
				return nil, nil
			}
			if ln != nil {
				_ = ln.Close()
			}
			for _, c := range conns {
				_ = c.Close()
			}
			return nil, nil
		})
}

func acceptLoop(rt *runtime.Runtime, types tcpTypes, state *listenerState, svcObj *values.Object) {
	for {
		conn, err := state.ln.Accept()
		if err != nil {
			return
		}
		state.mu.Lock()
		state.conns[conn] = struct{}{}
		state.mu.Unlock()
		go func() {
			defer func() {
				state.mu.Lock()
				delete(state.conns, conn)
				state.mu.Unlock()
			}()
			handleConnection(rt, types, svcObj, conn)
		}()
	}
}
