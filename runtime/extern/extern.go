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

	"ballerina/platform/pal"
	"ballerina/runtime/internal/locks"
	"ballerina/semtypes"
	"ballerina/values"
)

// NativeFunc is the signature for extern (native) function implementations.
// The first argument is the per-strand runtime context.
type NativeFunc func(ctx *Context, args []values.BalValue) (values.BalValue, error)

// Context represent per strand state
type Context struct {
	Env       *Env
	CallStack any // opaque pointer to the call stack
	TypeCtx   semtypes.Context
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

func CreateContext(env *Env) *Context {
	tyCtx := semtypes.ContextFrom(env.TypeEnv)
	ctx := Context{Env: env, TypeCtx: tyCtx, StrandID: env.AllocateStrandID()}
	return &ctx
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
