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

package runtime

import (
	"errors"

	"ballerina-lang-go/bir"
	"ballerina-lang-go/model"
	"ballerina-lang-go/platform/pal"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/runtime/internal/exec"
	"ballerina-lang-go/runtime/internal/modules"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

// LookupFunction resolves a top-level Ballerina function (BIR or native)
// by qualified name. The returned payload is opaque; pass it to
// InvokeFunction.
func LookupFunction(rt *Runtime, org, module, name string) (any, bool) {
	return exec.LookupFunction(rt.env, org, module, name)
}

func InvokeFunction(rt *Runtime, fn any, args []values.BalValue) (values.BalValue, error) {
	cx := exec.CreateContext(rt.env)
	return exec.Invoke(cx, fn, args)
}

const onGracefulStopLookupKey = "ballerina/lang.runtime:onGracefulStop"

func (rt *Runtime) runtimeBuiltins() map[string]extern.NativeFunc {
	return map[string]extern.NativeFunc{
		onGracefulStopLookupKey: rt.invokeOnGracefulStop,
	}
}

func (rt *Runtime) invokeOnGracefulStop(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	if len(args) != 1 {
		return nil, errors.New("lang.runtime:onGracefulStop expects one argument")
	}
	handler, ok := args[0].(*values.Function)
	if !ok {
		return nil, errors.New("lang.runtime:onGracefulStop expects a function")
	}
	handle, err := exec.NewFunctionValueHandle(rt.env, handler)
	if err != nil {
		return nil, err
	}
	if err := rt.registerGracefulStopHandler(handle); err != nil {
		return nil, err
	}
	return nil, nil
}

// Runtime represents a Ballerina runtime instance that owns a module registry
// and is used as the execution context for interpreting BIR packages.
//
// The embedded lifeCycle holds all lifecycle state machine fields; they are
// private to this package and mutated only via the methods in lifecycle.go.
type Runtime struct {
	lifeCycle
	env        *extern.Env
	ExitStatus <-chan uint8
}

// ModuleInitializer is a function that can install modules (e.g. stdlibs) into
// a runtime instance during its construction.
type ModuleInitializer func(*Runtime)

var moduleInitializers []ModuleInitializer

// NewRuntime constructs a new runtime with an empty registry and runs all
// registered module initializers.
func NewRuntime(platform pal.Platform, tyEnv semtypes.Env) *Runtime {
	exitChanel := make(chan uint8, 1)
	rt := &Runtime{
		ExitStatus: exitChanel,
		lifeCycle: lifeCycle{
			exitCodeChan: exitChanel,
		},
	}
	registry := modules.NewRegistry(rt.runtimeBuiltins())
	env := extern.InitEnv(platform, tyEnv, registry, extern.DispatchHandles{
		LookupObject:   exec.LookupObjectMethod,
		LookupRemote:   exec.LookupRemoteMethod,
		LookupResource: exec.LookupResourceMethod,
		Invoke:         exec.Invoke,
		Start:          exec.StartMethod,
		LookupFunction: func(cx *extern.Context, org, module, name string) (any, bool) {
			return exec.LookupFunction(cx.Env, org, module, name)
		},
	})
	rt.env = env
	for _, init := range moduleInitializers {
		init(rt)
	}
	return rt
}

// Platform returns the platform configuration of this runtime instance.
func (rt *Runtime) Platform() pal.Platform {
	return rt.env.Platform
}

func (rt *Runtime) registry() *modules.Registry {
	return rt.env.Registry.(*modules.Registry)
}

// Init registers and initializes a single BIR package. Callers must invoke
// Init in module-topological order. After every Init succeeds (or one
// fails), call Listen.
func (rt *Runtime) Init(pkg bir.BIRPackage) error {
	rt.transition(StateInitializing)
	rt.registry().RegisterModule(pkg.PackageID, modules.NewBIRModule(nil, &pkg))
	if err := rt.recordLifecycleHooks(&pkg); err != nil {
		return rt.abortInitialization(err)
	}
	if err := exec.RunEntrypoints(pkg, rt.env); err != nil {
		return rt.abortInitialization(err)
	}
	return nil
}

func (rt *Runtime) abortInitialization(err error) error {
	rt.mu.Lock()
	if rt.exitCode == 0 {
		rt.exitCode = 1
	}
	rt.mu.Unlock()
	rt.transition(StateGracefulStopping)
	return err
}

// recordLifecycleHooks appends the package's lifecycle dispatch handles onto the
// runtime's per-state slices in module-topological order. The three handles
// must be set together; partial population is a packager bug.
func (rt *Runtime) recordLifecycleHooks(pkg *bir.BIRPackage) error {
	hasAny := pkg.StartFunction != nil || pkg.GracefulStopFunction != nil || pkg.ImmediateStopFunction != nil
	if !hasAny {
		return nil
	}
	if pkg.StartFunction == nil || pkg.GracefulStopFunction == nil || pkg.ImmediateStopFunction == nil {
		return errors.New("malformed package lifecycle hooks: $start/$gracefulStop/$immediateStop must be set together")
	}
	rt.startFns = append(rt.startFns, exec.NewBIRHandle(pkg.StartFunction))
	rt.gracefulStopFns = append(rt.gracefulStopFns, exec.NewBIRHandle(pkg.GracefulStopFunction))
	rt.immediateStopFns = append(rt.immediateStopFns, exec.NewBIRHandle(pkg.ImmediateStopFunction))
	return nil
}

// Listen transitions the runtime into the Listening state. If no $start
// hooks have been registered the runtime moves straight to Stopped.
func (rt *Runtime) Listen() {
	rt.mu.Lock()
	stopped := rt.state == StateStopped
	hasListeners := len(rt.startFns) > 0
	rt.mu.Unlock()
	if stopped {
		return
	}
	if !hasListeners {
		rt.transition(StateGracefulStopping)
		return
	}
	rt.transition(StateListening)
}

// RegisterModuleInitializer registers a module initializer that will be invoked
// for every newly created runtime.
func RegisterModuleInitializer(init ModuleInitializer) {
	moduleInitializers = append(moduleInitializers, init)
}

// GetTypeEnv returns the semantic type environment.
func (rt *Runtime) GetTypeEnv() semtypes.Env {
	return rt.env.TypeEnv
}

// RegisterExternFunction registers a native (extern) function implementation in
// the given runtime instance so it can be called from interpreted BIR code.
func RegisterExternFunction(rt *Runtime, orgName string, moduleName string, funcName string, impl extern.NativeFunc) {
	rt.registry().RegisterExternFunction(orgName, moduleName, funcName, impl)
}

// RegisterExternClassDef registers a synthetic BIRClassDef for a Go-declared class so
// that execNewObject can resolve it. VTable entries have no BIR body; exec falls through
// to nativeFunctions for method dispatch.
func RegisterExternClassDef(rt *Runtime, def *bir.BIRClassDef) {
	rt.registry().RegisterExternClassDef(def)
}

// RegisterModuleGlobals makes module-level constants accessible at runtime.
// When Ballerina source code accesses an extern package's constant (e.g. http:LEADING),
// the BIR executor looks it up as a global variable in that package's module. Without
// registration, GetModule returns nil and causes a nil dereference panic.
func RegisterModuleGlobals(rt *Runtime, pkgId *model.PackageID, globals map[string]values.BalValue) {
	if existing := rt.registry().GetModule(pkgId); existing != nil {
		if existing.Globals == nil {
			existing.Globals = make(map[string]values.BalValue)
		}
		for k, v := range globals {
			existing.Globals[k] = v
		}
		return
	}
	rt.registry().RegisterModule(pkgId, &modules.BIRModule{Globals: globals})
}
