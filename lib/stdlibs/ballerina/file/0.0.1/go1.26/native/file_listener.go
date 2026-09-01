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
	"fmt"
	"os"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// fileListenerState is the Go-side state of a file:Listener object, stored on
// the object's "$state" field. The OS-level watch is created lazily in
// Listener.start (driven by the module's $start lifecycle hook, same as
// http:Listener); attach only registers services.
type fileListenerState struct {
	mu        sync.Mutex
	path      string
	recursive bool
	services  []*values.Object
	watch     pal.WatchHandle
}

// fileEventOpNames maps a pal.WatchOp to the FileEvent.operation string.
var fileEventOpNames = map[pal.WatchOp]string{
	pal.WatchCreate: "create",
	pal.WatchModify: "modify",
	pal.WatchDelete: "delete",
}

// fileEventRemoteMethodNames maps a pal.WatchOp to the remote method a
// service must declare to observe it.
var fileEventRemoteMethodNames = map[pal.WatchOp]string{
	pal.WatchCreate: "onCreate",
	pal.WatchModify: "onModify",
	pal.WatchDelete: "onDelete",
}

// newFileEventType builds the FileEvent record shape. Built once per runtime
// in initFileListenerModule, before the runtime freezes its type env — see
// fileTypes in file.go for why this must be per-runtime, not a package var.
func newFileEventType(env semtypes.Env) semtypes.SemType {
	md := semtypes.NewMappingDefinition()
	return md.Define(env, nil, semtypes.String)
}

func initFileListenerModule(rt *runtime.Runtime) {
	eventTy := newFileEventType(rt.GetTypeEnv())

	runtime.RegisterExternFunction(rt, orgName, moduleName, "initListener",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			cfg, _ := args[1].(*values.Map)
			path := ""
			recursive := false
			if cfg != nil {
				if v, ok := cfg.Get("path"); ok {
					path, _ = v.(string)
				}
				if v, ok := cfg.Get("recursive"); ok {
					recursive, _ = v.(bool)
				}
			}
			if path == "" {
				return fileError("FileSystemError", "'path' field is empty"), nil
			}
			info, err := rt.Platform().FS.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					return fileError("FileSystemError", "Folder does not exist: "+path), nil
				}
				return fileError("FileSystemError", err.Error()), nil
			}
			if !info.IsDir {
				return fileError("FileSystemError", "Unable to find a directory: "+path), nil
			}
			self.Put("$state", &fileListenerState{path: path, recursive: recursive})
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "registerListener",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			svcObj, ok := args[1].(*values.Object)
			if !ok {
				return values.NewErrorWithMessage("Listener attach: expected a service object"), nil
			}
			if !hasFileEventRemoteMethod(svcObj) {
				return fileError("GenericError", "At least a single resource required from following: "+
					"onCreate ,onDelete ,onModify. Parameter should be of type - file:FileEvent"), nil
			}
			state, ok := listenerState(self)
			if !ok {
				return values.NewErrorWithMessage("Listener attach: listener not initialised"), nil
			}
			state.mu.Lock()
			state.services = append(state.services, svcObj)
			state.mu.Unlock()
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "deregisterListener",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			svcObj, _ := args[1].(*values.Object)
			state, ok := listenerState(self)
			if !ok {
				return nil, nil
			}
			state.mu.Lock()
			defer state.mu.Unlock()
			for i, s := range state.services {
				if s == svcObj {
					state.services = append(state.services[:i], state.services[i+1:]...)
					break
				}
			}
			return nil, nil
		})

	// Listener.start creates the OS-level directory watch. It is invoked by
	// the module's $start lifecycle hook after all services have been
	// attached.
	runtime.RegisterExternFunction(rt, orgName, moduleName, "startListener",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			state, ok := listenerState(self)
			if !ok {
				return values.NewErrorWithMessage("Listener start: listener not initialised"), nil
			}
			state.mu.Lock()
			alreadyStarted := state.watch != nil
			state.mu.Unlock()
			if alreadyStarted {
				return nil, nil
			}
			handle, err := rt.Platform().FS.Watch(state.path, state.recursive, func(ev pal.WatchEvent) {
				dispatchFileEvent(rt, state, eventTy, ev)
			})
			if err != nil {
				return fileError("FileSystemError", "Unable to initialize server connector: "+err.Error()), nil
			}
			state.mu.Lock()
			state.watch = handle
			state.mu.Unlock()
			return nil, nil
		})

	// stopListener backs both gracefulStop and immediateStop. Unlike
	// jBallerina — whose gracefulStop is a no-op that leaves the watch
	// thread running until process exit — this closes the OS watch
	// deterministically on either call, since a single long-lived process
	// here may create and stop many listeners (e.g. across test cases).
	runtime.RegisterExternFunction(rt, orgName, moduleName, "stopListener",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			state, ok := listenerState(self)
			if !ok {
				return nil, nil
			}
			state.mu.Lock()
			watch := state.watch
			state.watch = nil
			state.mu.Unlock()
			if watch != nil {
				_ = watch.Close()
			}
			return nil, nil
		})
}

func listenerState(self *values.Object) (*fileListenerState, bool) {
	stateVal, ok := self.Get("$state")
	if !ok {
		return nil, false
	}
	state, ok := stateVal.(*fileListenerState)
	return state, ok
}

// hasFileEventRemoteMethod reports whether svcObj declares at least one of
// onCreate/onModify/onDelete, matching jBallerina's directory-listener
// attach-time validation.
func hasFileEventRemoteMethod(svcObj *values.Object) bool {
	for _, name := range fileEventRemoteMethodNames {
		if _, ok := svcObj.MethodLookupKey(model.RemoteMethodName(name)); ok {
			return true
		}
	}
	return false
}

// dispatchFileEvent invokes the matching remote method on every attached
// service for a single filesystem event. Runs on the watch's own goroutine,
// independent of any strand that called start/attach.
//
// A panic in one service's remote method aborts dispatch to the remaining
// services for this event only (see recoverFileEventPanic) — the next event
// dispatches normally on a freshly reset context.
func dispatchFileEvent(rt *runtime.Runtime, state *fileListenerState, eventTy semtypes.SemType, ev pal.WatchEvent) {
	methodName, ok := fileEventRemoteMethodNames[ev.Op]
	if !ok {
		return
	}
	state.mu.Lock()
	services := make([]*values.Object, len(state.services))
	copy(services, state.services)
	state.mu.Unlock()
	if len(services) == 0 {
		return
	}
	ctx := rt.AcquirePooledContext()
	// Release the pool slot last (LIFO): recoverFileEventPanic must run first,
	// while ctx is still live and un-reset, so it can drain any lock left held
	// by a panic mid-lock-block before the context goes back to the pool.
	defer rt.ReleasePooledContext(ctx)
	defer recoverFileEventPanic(rt, ctx, methodName, ev)
	eventVal := buildFileEvent(ctx, eventTy, ev)
	for _, svcObj := range services {
		handle, ok := ctx.LookupRemoteMethod(svcObj, methodName)
		if !ok {
			continue
		}
		_, _ = ctx.InvokeMethod(handle, []values.BalValue{svcObj, eventVal})
	}
}

// recoverFileEventPanic recovers a panic raised while dispatching a file
// event, so a single misbehaving service doesn't crash the whole process
// (this runs on the fsnotify watcher's dedicated goroutine, which has no
// other recover on its call stack). Releases any lock ctx still holds first:
// a Go-level panic (e.g. divide by zero) inside a Ballerina lock block skips
// the matching LockEnd, and ctx's pooled-reuse reset does not itself unlock —
// it only clears the held-lock bookkeeping — so skipping this would strand
// the lock and deadlock every later event that touches the same lock
// variable on this listener's single dispatch goroutine.
func recoverFileEventPanic(rt *runtime.Runtime, ctx *extern.Context, methodName string, ev pal.WatchEvent) {
	rec := recover()
	if rec == nil {
		return
	}
	ctx.ReleaseAllHeldLocks()
	logMsg := fmt.Sprintf("error [ballerina/file]: panic while dispatching %s for %s: %s\n",
		methodName, ev.Path, panicMessage(rec))
	_, _ = rt.Platform().IO.Stderr([]byte(logMsg))
}

// panicMessage renders a recovered panic value as a message string.
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

func buildFileEvent(ctx *extern.Context, eventTy semtypes.SemType, ev pal.WatchEvent) *values.Map {
	atomic := semtypes.ToMappingAtomicType(ctx.TypeCtx(), eventTy)
	return values.NewMap(eventTy, atomic, false, []values.MapEntry{
		{Key: "name", Value: ev.Path},
		{Key: "operation", Value: fileEventOpNames[ev.Op]},
	})
}

func init() {
	runtime.RegisterModuleInitializer(initFileListenerModule)
}
