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
	"math"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	listenerStateField = "$listener"
	negativeInterval   = "Repeat interval must be >= 0"
	zeroInterval       = "Repeat Interval cannot be zero."
)

// emailListener is the state of an email:PopListener or email:ImapListener: the client it
// polls and the services it dispatches to.
type emailListener struct {
	rt       *runtime.Runtime
	client   *values.Object
	mailbox  *mailbox
	interval *decimal.Decimal

	// dispatching is the context running a service method, so a stop from it does not wait for it.
	dispatching atomic.Pointer[extern.Context]

	mu       sync.Mutex
	services []*values.Object
	poller   *poller
	// last is the most recent poller, which may still be running after a stop that did not wait.
	last *poller
}

// poller is one run of the polling loop, from start to stop.
type poller struct {
	stop chan struct{}
	done chan struct{}
	// closeOnExit is set by a stop that does not wait, so the loop calls onClose when it ends.
	closeOnExit atomic.Bool
}

func initListenerExtern(rt *runtime.Runtime) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		client, _ := args[1].(*values.Object)
		interval, _ := args[2].(*decimal.Decimal)
		m := mailboxOf(client, popStateField)
		if m == nil {
			m = mailboxOf(client, imapStateField)
		}
		args[0].(*values.Object).Put(listenerStateField, &emailListener{rt: rt, client: client, mailbox: m, interval: interval})
		return nil, nil
	}
}

func registerServiceExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	l := listenerOf(args[0])
	svc := args[1].(*values.Object)
	l.mu.Lock()
	defer l.mu.Unlock()
	if !slices.Contains(l.services, svc) {
		l.services = append(l.services, svc)
	}
	return nil, nil
}

func deregisterServiceExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	l := listenerOf(args[0])
	svc := args[1].(*values.Object)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.services = slices.DeleteFunc(l.services, func(s *values.Object) bool { return s == svc })
	return nil, nil
}

// startListenerExtern starts polling every pollingInterval seconds, validated as the task
// scheduler jBallerina uses does; starting a running listener does nothing.
func startListenerExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	l := listenerOf(args[0])
	millis := intervalMillis(l.interval)
	if millis < 0 {
		return emailError(negativeInterval), nil
	}
	if millis == 0 {
		return emailError(zeroInterval), nil
	}
	interval := time.Duration(min(millis, math.MaxInt64/int64(time.Millisecond))) * time.Millisecond
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.poller != nil {
		return nil, nil
	}
	p := &poller{stop: make(chan struct{}), done: make(chan struct{})}
	prev := l.last
	l.poller, l.last = p, p
	go l.run(p, prev, interval)
	return nil, nil
}

// stopListenerExtern stops polling. A graceful stop waits for the running poll and then calls
// onClose; other stops return at once and the polling loop calls onClose when it ends.
func stopListenerExtern(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	l := listenerOf(args[0])
	graceful, _ := args[1].(bool)
	l.mu.Lock()
	p, last := l.poller, l.last
	l.poller = nil
	l.mu.Unlock()
	wait := graceful && l.dispatching.Load() != ctx
	if p != nil {
		p.closeOnExit.Store(!wait)
		close(p.stop)
	}
	if wait && last != nil {
		for !closed(last.done) {
			<-ctx.Yield()
		}
	}
	if wait && p != nil {
		l.closeServices()
	}
	return nil, nil
}

// run polls until stopped, once the previous poller has finished.
func (l *emailListener) run(p, prev *poller, interval time.Duration) {
	defer close(p.done)
	if prev != nil {
		<-prev.done
	}
	for !closed(p.stop) {
		l.drain(p)
		fired := make(chan struct{})
		timer := l.rt.Platform().Time.AfterFunc(interval, func() { close(fired) })
		select {
		case <-fired:
		case <-p.stop:
			timer.Stop()
		}
	}
	if p.closeOnExit.Load() {
		l.closeServices()
	}
}

// drain dispatches messages until the folder is empty, an error is dispatched, a message stays
// in the folder or the listener stops; it reads only when a service declares onMessage.
func (l *emailListener) drain(p *poller) {
	for !closed(p.stop) {
		services := l.attached()
		if !l.declares(services, "onMessage") {
			return
		}
		result := l.receive()
		if result == nil {
			return
		}
		readErr, isErr := result.(*values.Error)
		method := "onMessage"
		if isErr {
			method = "onError"
		}
		handled := false
		for _, svc := range services {
			handled = l.call(svc, method, result) || handled
		}
		if isErr {
			if !handled {
				l.logError(readErr.Message)
			}
			return
		}
		// A message that could not be deleted or marked seen would be read again at once.
		if l.mailbox.kept.Load() {
			return
		}
	}
}

func (l *emailListener) closeServices() {
	for _, svc := range l.attached() {
		l.call(svc, "onClose", nil)
	}
}

// receive reads one message with the client's receiveMessage, so errors are email:Error values.
func (l *emailListener) receive() values.BalValue {
	ctx := l.rt.AcquirePooledContext()
	defer l.rt.ReleasePooledContext(ctx)
	h, _ := ctx.LookupRemoteMethod(l.client, "receiveMessage")
	result, err := ctx.InvokeMethod(h, []values.BalValue{l.client, defaultFolder, decimal.FromInt64(0)})
	if err != nil {
		l.logError(err.Error())
		return nil
	}
	return result
}

func (l *emailListener) declares(services []*values.Object, method string) bool {
	ctx := l.rt.AcquirePooledContext()
	defer l.rt.ReleasePooledContext(ctx)
	return slices.ContainsFunc(services, func(svc *values.Object) bool {
		_, ok := ctx.LookupRemoteMethod(svc, method)
		return ok
	})
}

// call invokes method on svc, if declared, on its own context; a panic or a returned error is
// logged. It reports whether the service declares the method.
func (l *emailListener) call(svc *values.Object, method string, arg values.BalValue) bool {
	ctx := l.rt.AcquirePooledContext()
	defer l.rt.ReleasePooledContext(ctx)
	h, ok := ctx.LookupRemoteMethod(svc, method)
	if !ok {
		return false
	}
	l.dispatching.Store(ctx)
	defer l.dispatching.Store(nil)
	defer func() {
		if rec := recover(); rec != nil {
			ctx.ReleaseAllHeldLocks()
			l.logError(panicMessage(rec))
		}
	}()
	result, err := ctx.InvokeMethod(h, []values.BalValue{svc, arg})
	if err != nil {
		l.logError(err.Error())
	} else if returned, isErr := result.(*values.Error); isErr {
		l.logError(returned.Message)
	}
	return true
}

func (l *emailListener) attached() []*values.Object {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.services)
}

func (l *emailListener) logError(message string) {
	_, _ = l.rt.Platform().IO.Stderr([]byte("error: " + message + "\n"))
}

func listenerOf(v values.BalValue) *emailListener {
	state, _ := v.(*values.Object).Get(listenerStateField)
	return state.(*emailListener)
}

func closed(ch chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// intervalMillis converts seconds to milliseconds, truncating toward zero and saturating at the
// int64 range.
func intervalMillis(seconds *decimal.Decimal) int64 {
	negative := seconds.Cmp(decimal.FromInt64(0)) < 0
	limit := int64(math.MaxInt64)
	if negative {
		limit = math.MinInt64
	}
	ms, err := seconds.Mul(decimal.FromInt64(1000))
	if err != nil {
		return limit
	}
	round := ms.Floor
	if negative {
		round = ms.Ceiling
	}
	whole, err := round()
	if err != nil {
		return limit
	}
	n, fits, _ := whole.Int64()
	if !fits {
		return limit
	}
	return n
}

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
