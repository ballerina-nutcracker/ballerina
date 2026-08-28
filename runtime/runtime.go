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
	"sync"

	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/runtime/internal/exec"
	"github.com/ballerina-nutcracker/ballerina/runtime/internal/modules"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
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
	// ctxPool recycles extern.Contexts across invocations so the (program-
	// constant) semtype context caches stay warm instead of being reallocated
	// per call — a hot path for request-per-invocation workloads like http.
	ctxPool sync.Pool
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
		LookupObject:         exec.LookupObjectMethod,
		LookupRemote:         exec.LookupRemoteMethod,
		LookupResource:       exec.LookupResourceMethod,
		LookupResourceByPath: exec.LookupResourceMethodByPath,
		Invoke:               exec.Invoke,
		Start:                exec.StartMethod,
		LookupFunction: func(cx *extern.Context, org, module, name string) (any, bool) {
			return exec.LookupFunction(cx.Env, org, module, name)
		},
	}, extern.MetadataHandles{
		Signature:         exec.FunctionSignature,
		Metadata:          exec.FunctionMetadata,
		ObjectAnnotations: exec.ObjectAnnotations,
		TypeAnnotations:   exec.TypeAnnotations,
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
	rt.registry().RegisterModule(pkg.PackageID, modules.NewBIRModule(semtypes.ContextFrom(rt.env.TypeEnv), &pkg))
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

// AcquirePooledContext creates a properly initialised extern.Context with a
// fresh call stack, reusing a pooled context when one is available. Use this
// only when the caller knows exactly when its unit of work ends and can
// guarantee a matching ReleasePooledContext call (e.g. HTTP resource/remote
// dispatch, which releases once the response is fully written) — every
// context obtained here MUST be released back to the pool. Callers that
// cannot make that guarantee (starting a strand, the public calling API)
// should use exec.CreateContext / extern.CreateContext instead; going through
// the pool without a guaranteed release point is just overhead, and a context
// that never gets released is never reused.
//
// A context handed out here is already reset for a fresh unit of work —
// ReleasePooledContext resets it before returning it to the pool, so it never
// sits idle holding a prior request's state.
func (rt *Runtime) AcquirePooledContext() *extern.Context {
	if v := rt.ctxPool.Get(); v != nil {
		return v.(*extern.Context)
	}
	return exec.CreateContext(rt.env)
}

// ReleasePooledContext returns a context obtained from AcquirePooledContext to
// the pool for reuse. Call it once the owning strand is done with the context
// and nothing else still references it or its TypeCtx (e.g. after the
// response is fully written).
//
// Resets the context before pooling it — rather than on the next Acquire —
// so a context sitting idle in the pool doesn't keep a finished request's
// call-stack frames, TypeCtx memo caches, and held-lock slots reachable any
// longer than necessary.
//
// Async work spawned during the invocation is safe: StartMethod snapshots the
// caller's frames by value and runStrand builds each started strand its own
// context via CreateContext (own TypeCtx), so a started strand never aliases
// this context — releasing it does not race with in-flight children. The only
// shared state is the program-wide Env, which every context already shares and
// which is not recycled here.
func (rt *Runtime) ReleasePooledContext(ctx *extern.Context) {
	exec.ResetContextForReuse(ctx)
	rt.ctxPool.Put(ctx)
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
