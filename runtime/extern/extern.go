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

// Package extern represents runtime API that is visible to extern functions.
package extern

import (
	"sync/atomic"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime/internal/locks"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// NativeFunc is the signature for extern (native) function implementations.
// The first argument is the per-strand runtime context.
type NativeFunc func(ctx *Context, args []values.BalValue) (values.BalValue, error)

// Context represent per strand state
type Context struct {
	Env       *Env
	CallStack any // opaque pointer to the call stack
	typeCtx   semtypes.Context
	StrandID  uint64
	heldLocks []*locks.ReentrantMutex
}

// Env represents shared state of the runtime. All operations on Env are (potentially) blocking
type Env struct {
	Platform     pal.Platform
	TypeEnv      semtypes.Env
	Registry     any // opaque pointer to the runtime registry
	Locks        *locks.LockManager
	dispatch     DispatchHandles
	nextStrandID atomic.Uint64
}

func InitEnv(pal pal.Platform, tyEnv semtypes.Env, registry any, hooks DispatchHandles) *Env {
	env := Env{
		Platform: pal,
		TypeEnv:  tyEnv,
		Registry: registry,
		Locks:    locks.NewMutexes(),
		dispatch: hooks,
	}
	return &env
}

// AllocateStrandID returns a fresh, non-zero strand id. Zero is reserved as
// the unowned-mutex sentinel.
func (e *Env) AllocateStrandID() uint64 {
	for {
		v := e.nextStrandID.Add(1)
		if v != 0 {
			return v
		}
	}
}

// CreateContext builds a fresh, unpooled Context. Use this for callers that
// cannot guarantee a single clear release point — e.g. a strand started via
// StartMethod, or the public InvokeFunction API — where routing through a
// pool would be pure overhead (or a correctness risk if the context outlives
// its release). Callers that own a well-defined unit of work end-to-end (like
// HTTP resource/remote dispatch) should prefer Runtime.AcquirePooledContext.
func CreateContext(env *Env) *Context {
	ctx := Context{Env: env, StrandID: env.AllocateStrandID()}
	return &ctx
}

// TypeCtx returns the strand's semtype type-check context, building it lazily
// on first use. Not every invocation performs a type check, so the context
// (and its memo caches) is only allocated for strands that actually call this.
func (ctx *Context) TypeCtx() semtypes.Context {
	if ctx.typeCtx == nil {
		ctx.typeCtx = semtypes.ContextFrom(ctx.Env.TypeEnv)
	}
	return ctx.typeCtx
}

// TypeEnv returns the program-constant semantic type environment directly,
// for callers that only need type definitions and not a full type-check
// context (which would otherwise force allocation of TypeCtx's memo caches).
func (ctx *Context) TypeEnv() semtypes.Env {
	return ctx.Env.TypeEnv
}

// ResetForReuse clears a context's per-request mutable state — a fresh strand
// ID, an empty held-lock stack, and (if one was ever built) a cleared TypeCtx
// — while retaining the constant Env. Clearing TypeCtx rather than keeping its
// memo caches warm bounds its memory across the pooled context's lifetime;
// combined with TypeCtx's lazy construction, an invocation that never
// type-checks pays nothing here either. Safe only under exclusive ownership
// (e.g. via sync.Pool).
func (ctx *Context) ResetForReuse() {
	ctx.StrandID = ctx.Env.AllocateStrandID()
	ctx.heldLocks = ctx.heldLocks[:0]
	if ctx.typeCtx != nil {
		ctx.typeCtx.Reset()
	}
}

// AcquireLock acquires the global re-entrant mutex for the given lock key on
// behalf of the current strand and pushes it onto the held-lock stack.
//
// Callers must pair every acquisition with a `LockEnd` BIR terminator (which
// invokes ReleaseLock). BIR-gen guarantees this for every abrupt-exit BIR
// terminator inside a `lock` body (Return / Break / Continue / Panic), but
// not for Go-level `panic`s raised below BIR (e.g. div-by-zero, array OOB).
// A `trap` that recovers such a Go-level panic does NOT release the lock
// — see the package doc on runtime/internal/locks.
func (ctx *Context) AcquireLock(key string) {
	m := ctx.Env.Locks.Get(key)
	m.Lock(ctx.StrandID)
	ctx.heldLocks = append(ctx.heldLocks, m)
}

// ReleaseLock pops the top entry of the held-lock stack and releases it.
func (ctx *Context) ReleaseLock() {
	n := len(ctx.heldLocks)
	top := ctx.heldLocks[n-1]
	ctx.heldLocks[n-1] = nil
	ctx.heldLocks = ctx.heldLocks[:n-1]
	top.Unlock(ctx.StrandID)
}

// ReleaseAllHeldLocks drains the held-lock stack in LIFO order.
//
// This is the last-resort uncaught-panic path: it is invoked by the
// interpreter's top-level recover so a strand never strands a held lock when
// the program aborts. `trap`-recovered panics do NOT invoke this — by design
// at this stage, locks recovered through `trap` remain owned by the strand
// (the lock is re-entrant per strand, so this is harmless within the same
// strand).
func (ctx *Context) ReleaseAllHeldLocks() {
	for i := len(ctx.heldLocks) - 1; i >= 0; i-- {
		ctx.heldLocks[i].Unlock(ctx.StrandID)
		ctx.heldLocks[i] = nil
	}
	ctx.heldLocks = ctx.heldLocks[:0]
}
