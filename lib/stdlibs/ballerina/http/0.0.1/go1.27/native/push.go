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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	promiseSlot        = "$promise"
	pushHTTP1          = "Sending a PUSH_PROMISE is not supported for HTTP/1.x connections"
	pushDisabled       = "Server push not allowed to opposite endpoint"
	invalidPromise     = "invalid push promise"
	promiseInvalidated = "Promised stream is already rejected or stream is no longer valid"
)

type headerField struct {
	name, value string
}

// pushPromise is the native state of an http:PushPromise: its headers, and the streams promised
// with it that wait for their response.
type pushPromise struct {
	mu      sync.Mutex
	headers []headerField
	// created marks the promise as used by promise or a header method, after which, as in
	// jBallerina, pushPromisedResponse no longer panics for it.
	created bool
	streams []*pushedStream
}

func promiseOf(obj *values.Object) *pushPromise {
	p, _ := obj.Get(promiseSlot)
	return p.(*pushPromise)
}

func (p *pushPromise) values(name string) []string {
	var found []string
	for _, f := range p.headers {
		if strings.EqualFold(f.name, name) {
			found = append(found, f.value)
		}
	}
	return found
}

func (p *pushPromise) remove(name string) {
	p.headers = slices.DeleteFunc(p.headers, func(f headerField) bool { return strings.EqualFold(f.name, name) })
}

// names returns each header name in the case it was first added, ordered case-insensitively as
// jBallerina's ExternPushPromise does.
func (p *pushPromise) names() []values.BalValue {
	var names []string
	for _, f := range p.headers {
		if !slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(n, f.name) }) {
			names = append(names, f.name)
		}
	}
	slices.SortStableFunc(names, func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) })
	items := make([]values.BalValue, len(names))
	for i, n := range names {
		items[i] = n
	}
	return items
}

func (p *pushPromise) header() http.Header {
	header := http.Header{}
	for _, f := range p.headers {
		header.Add(f.name, f.value)
	}
	return header
}

func (p *pushPromise) drop(s *pushedStream) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.streams = slices.DeleteFunc(p.streams, func(other *pushedStream) bool { return other == s })
}

type streamState uint8

const (
	streamWaiting streamState = iota
	streamClaimed
	streamCancelled
)

// pushedStream is a stream promised to the client. net/http dispatches its request to the
// listener, whose handler waits until the resource answers the promise or the stream is cancelled.
type pushedStream struct {
	conn    *callerConn
	promise *pushPromise
	key     string
	// arrived is closed once the handler of the pushed request has set w and r.
	arrived chan struct{}
	w       *responseEncoder
	r       *http.Request
	// done is closed once the stream is answered or cancelled.
	done  chan struct{}
	mu    sync.Mutex
	state streamState
}

// claim reserves the stream for a response and reports whether it was still waiting for one.
func (s *pushedStream) claim() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != streamWaiting {
		return false
	}
	s.state = streamClaimed
	return true
}

// cancel ends a stream no response was claimed for, which its handler then resets, and reports
// whether it did.
func (s *pushedStream) cancel() bool {
	s.mu.Lock()
	if s.state != streamWaiting {
		s.mu.Unlock()
		return false
	}
	s.state = streamCancelled
	close(s.done)
	s.mu.Unlock()
	s.promise.drop(s)
	return true
}

func (s *pushedStream) cancelled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state == streamCancelled
}

// pushRegistry holds the promised streams whose requests net/http has yet to dispatch, at most one
// per connection, method and target, so each pushed request finds the stream it was promised for.
type pushRegistry struct {
	mu sync.Mutex
	// taken is signalled when a waiting stream leaves the registry.
	taken   *sync.Cond
	waiting map[string]*pushedStream
}

func newPushRegistry() *pushRegistry {
	g := &pushRegistry{waiting: map[string]*pushedStream{}}
	g.taken = sync.NewCond(&g.mu)
	return g
}

func pushKey(remoteAddr, method, target string) string {
	return remoteAddr + " " + method + " " + target
}

// add registers s once no other stream waits under its key.
func (g *pushRegistry) add(s *pushedStream) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for g.waiting[s.key] != nil {
		g.taken.Wait()
	}
	g.waiting[s.key] = s
}

func (g *pushRegistry) withdraw(s *pushedStream) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.waiting[s.key] == s {
		delete(g.waiting, s.key)
		g.taken.Broadcast()
	}
}

// take returns the promised stream r is the pushed request of, if any. net/http does not mark a
// pushed request, so a request of the client for the same target on the same connection, arriving
// between promise and the dispatch of the pushed request, would be taken in its place.
func (g *pushRegistry) take(r *http.Request) *pushedStream {
	if r.ProtoMajor != 2 {
		return nil
	}
	key := pushKey(r.RemoteAddr, r.Method, r.URL.RequestURI())
	g.mu.Lock()
	defer g.mu.Unlock()
	s := g.waiting[key]
	if s != nil {
		delete(g.waiting, key)
		g.taken.Broadcast()
	}
	return s
}

// servePush answers r when it is the request of a promised stream, once the resource sends the
// promised response, and resets the stream when it is cancelled or the client resets it first.
func servePush(state *listenerState, w *responseEncoder, r *http.Request) bool {
	s := state.pushes.take(r)
	if s == nil {
		return false
	}
	w.useService(s.conn.config)
	s.w, s.r = w, r
	close(s.arrived)
	select {
	case <-s.done:
	case <-r.Context().Done():
		if !s.cancel() {
			<-s.done
		}
	}
	if s.cancelled() {
		w.abort()
	}
	return true
}

// promise sends a PUSH_PROMISE for promise on the stream of the request. As in jBallerina, once
// the response is sent nothing more is promised.
func (c *callerConn) promise(ctx *extern.Context, obj *values.Object) values.BalValue {
	p := promiseOf(obj)
	path, _ := obj.Get("path")
	method, _ := obj.Get("method")
	p.mu.Lock()
	p.created = true
	header := p.header()
	p.mu.Unlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	pusher, ok := c.w.(*responseEncoder).ResponseWriter.(http.Pusher)
	if !ok || c.r.ProtoMajor != 2 {
		return httpError(ctx, genericListenerError, pushHTTP1, nil)
	}
	if c.closed() {
		return nil
	}
	target, err := promisedTarget(path.(string), method.(string))
	if err != nil {
		return httpError(ctx, genericListenerError, err.Error(), nil)
	}
	s := &pushedStream{conn: c, promise: p, key: pushKey(c.r.RemoteAddr, method.(string), target),
		arrived: make(chan struct{}), done: make(chan struct{})}
	pushes := c.exchange.state.pushes
	pushes.add(s)
	if err := pusher.Push(path.(string), &http.PushOptions{Method: method.(string), Header: header}); err != nil {
		// net/http may have started the pushed request's handler before failing.
		pushes.withdraw(s)
		s.cancel()
		// Unlike jBallerina, nothing is sent to a client that disabled push (#9919).
		if errors.Is(err, http.ErrNotSupported) {
			return httpError(ctx, genericListenerError, pushDisabled, nil)
		}
		return httpError(ctx, genericListenerError, err.Error(), nil)
	}
	// net/http starts the handler of the pushed request before Push returns; waiting for it keeps
	// the next promise for the same target from being bound to this request.
	<-s.arrived
	c.pushes = append(c.pushes, s)
	p.mu.Lock()
	p.streams = append(p.streams, s)
	p.mu.Unlock()
	return nil
}

// promisedTarget returns the request target of a promise for path and method. Unlike jBallerina,
// which sends them, a relative or malformed path and a method other than GET and HEAD are rejected (#9921).
func promisedTarget(path, method string) (string, error) {
	if !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("invalid push promise path: '%s': expected an absolute path", path)
	}
	if method != http.MethodGet && method != http.MethodHead {
		return "", fmt.Errorf("invalid push promise method: '%s': expected 'GET' or 'HEAD'", method)
	}
	target, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("invalid push promise path: '%s': %w", path, errors.Unwrap(err))
	}
	return target.RequestURI(), nil
}

// pushResponse sends resp on the oldest stream promised with promise on this exchange that has
// no response yet. Unlike jBallerina, which writes a response for a promise never sent on stream 0,
// that is an error (#9921).
func (c *callerConn) pushResponse(ctx *extern.Context, obj *values.Object, resp *values.Object) (values.BalValue, error) {
	p := promiseOf(obj)
	p.mu.Lock()
	if !p.created {
		p.mu.Unlock()
		return nil, errors.New(invalidPromise)
	}
	var s *pushedStream
	for i, candidate := range p.streams {
		if candidate.conn == c && candidate.claim() {
			s = candidate
			p.streams = slices.Delete(p.streams, i, i+1)
			break
		}
	}
	p.mu.Unlock()
	if s == nil {
		return httpError(ctx, genericListenerError, promiseInvalidated, nil), nil
	}
	defer close(s.done)
	if err := writeResponse(c.rt, ctx, s.w, s.r, resp); err != nil {
		return httpError(ctx, genericListenerError, err.Error(), nil), nil
	}
	return nil, nil
}

// hold marks a stage as using the exchange.
func (c *callerConn) hold() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stages++
}

// release ends a stage that used the exchange. Once the last one has, the streams promised on it
// that never got a response are reset, which jBallerina leaves open (#9920).
func (c *callerConn) release() {
	c.mu.Lock()
	c.stages--
	last := c.stages == 0
	streams := c.pushes
	if last {
		c.pushes = nil
	}
	c.mu.Unlock()
	if last {
		for _, s := range streams {
			s.cancel()
		}
	}
}

func registerPushExterns(rt *runtime.Runtime, types *httpTypes) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externPromise",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			return connOf(args[0]).promise(ctx, args[1].(*values.Object)), nil
		})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externPushPromisedResponse",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			return connOf(args[0]).pushResponse(ctx, args[1].(*values.Object), args[2].(*values.Object))
		})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externPromiseInit",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			args[0].(*values.Object).Put(promiseSlot, &pushPromise{})
			return nil, nil
		})
	registerPromiseHeaderExterns(rt, types)
}

func registerPromiseHeaderExterns(rt *runtime.Runtime, types *httpTypes) {
	withPromise := func(fn func(ctx *extern.Context, p *pushPromise, args []values.BalValue) (values.BalValue, error)) extern.NativeFunc {
		return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			p := promiseOf(args[0].(*values.Object))
			p.mu.Lock()
			defer p.mu.Unlock()
			p.created = true
			return fn(ctx, p, args)
		}
	}
	register := func(name string, fn func(ctx *extern.Context, p *pushPromise, args []values.BalValue) (values.BalValue, error)) {
		runtime.RegisterExternFunction(rt, orgName, moduleName, name, withPromise(fn))
	}
	register("externPromiseHasHeader", func(_ *extern.Context, p *pushPromise, args []values.BalValue) (values.BalValue, error) {
		return len(p.values(args[1].(string))) > 0, nil
	})
	register("externPromiseGetHeader", func(_ *extern.Context, p *pushPromise, args []values.BalValue) (values.BalValue, error) {
		found := p.values(args[1].(string))
		if len(found) == 0 {
			return nil, errors.New("incompatible types: '()' cannot be cast to 'string'")
		}
		return found[0], nil
	})
	register("externPromiseGetHeaders", func(ctx *extern.Context, p *pushPromise, args []values.BalValue) (values.BalValue, error) {
		found := p.values(args[1].(string))
		items := make([]values.BalValue, len(found))
		for i, v := range found {
			items[i] = v
		}
		return newTypedListValue(ctx.TypeEnv(), types.strArrTy, items), nil
	})
	register("externPromiseAddHeader", func(_ *extern.Context, p *pushPromise, args []values.BalValue) (values.BalValue, error) {
		p.headers = append(p.headers, headerField{args[1].(string), args[2].(string)})
		return nil, nil
	})
	register("externPromiseSetHeader", func(_ *extern.Context, p *pushPromise, args []values.BalValue) (values.BalValue, error) {
		p.remove(args[1].(string))
		p.headers = append(p.headers, headerField{args[1].(string), args[2].(string)})
		return nil, nil
	})
	register("externPromiseRemoveHeader", func(_ *extern.Context, p *pushPromise, args []values.BalValue) (values.BalValue, error) {
		p.remove(args[1].(string))
		return nil, nil
	})
	register("externPromiseRemoveAllHeaders", func(_ *extern.Context, p *pushPromise, _ []values.BalValue) (values.BalValue, error) {
		p.headers = nil
		return nil, nil
	})
	register("externPromiseGetHeaderNames", func(ctx *extern.Context, p *pushPromise, _ []values.BalValue) (values.BalValue, error) {
		return newTypedListValue(ctx.TypeEnv(), types.strArrTy, p.names()), nil
	})
}
