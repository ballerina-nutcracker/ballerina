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
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	illegalRespond = "illegal respond: response has already been sent"
	illegalReturn  = "illegal return: response has already been sent"
)

// callerConn is the exchange an http:Caller answers. A request gets one response: the first
// respond sends it, and once the resource has returned no respond can.
type callerConn struct {
	rt    *runtime.Runtime
	types *httpTypes
	w     http.ResponseWriter
	r     *http.Request
	// accessor is the dispatched resource accessor: a lower-case method or "default".
	accessor string
	config   serviceConfig

	mu    sync.Mutex
	state callerState
	// sent is closed once the response is complete; the handler waits on it and w must not be
	// written after it is closed.
	sent     chan struct{}
	sentOnce sync.Once
}

type callerState uint8

const (
	// stateResponded marks a final response as sent.
	stateResponded callerState = 1 << iota
	// stateFinished marks the resource as returned, after which nothing can be sent.
	stateFinished
	// stateBodyBound marks the request body as handed to a Request or payload parameter.
	stateBodyBound
)

func (s callerState) has(flag callerState) bool { return s&flag != 0 }

func (s *callerState) set(flag callerState) { *s |= flag }

// closed reports whether no response can be sent any more. The caller holds c.mu.
func (c *callerConn) closed() bool {
	return c.state.has(stateResponded) || c.state.has(stateFinished)
}

// close marks the exchange finished and reports whether a response was already sent.
func (c *callerConn) close() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.set(stateFinished)
	return c.state.has(stateResponded)
}

// markSent releases the handler waiting for the response.
func (c *callerConn) markSent() {
	c.sentOnce.Do(func() { close(c.sent) })
}

// bindBody records that a Request or payload parameter reads the request body.
func (c *callerConn) bindBody() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.set(stateBodyBound)
}

// newCaller builds the http:Caller a resource parameter receives for conn.
func newCaller(ctx *extern.Context, state *listenerState, conn *callerConn) (values.BalValue, error) {
	remoteHost, remotePort := splitHostPort(conn.r.RemoteAddr)
	var localHost string
	var localPort int64
	if addr, ok := conn.r.Context().Value(http.LocalAddrContextKey).(net.Addr); ok {
		localHost, localPort = splitHostPort(addr.String())
	}
	caller, err := ctx.InvokeFunction(state.createCaller, []values.BalValue{
		remoteHost, remotePort, remoteHost, localHost, localPort, localHost, strings.ToUpper(conn.accessor),
	})
	if err != nil {
		return nil, err
	}
	caller.(*values.Object).Put("$conn", conn)
	return caller, nil
}

// splitHostPort splits a "host:port" socket address, writing an IP the way Java's
// InetAddress.getHostAddress does.
func splitHostPort(address string) (string, int64) {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return address, 0
	}
	port, _ := strconv.ParseInt(portStr, 10, 64)
	return javaHostAddress(host), port
}

// javaHostAddress writes an IPv6 address as eight uncompressed hex groups
// ("0:0:0:0:0:0:0:1" for "::1") and leaves IPv4 addresses and names unchanged.
func javaHostAddress(host string) string {
	ipPart, zone, _ := strings.Cut(host, "%")
	ip := net.ParseIP(ipPart)
	if ip == nil || ip.To4() != nil {
		return host
	}
	groups := make([]string, 8)
	for i := range groups {
		groups[i] = strconv.FormatUint(uint64(ip[2*i])<<8|uint64(ip[2*i+1]), 16)
	}
	result := strings.Join(groups, ":")
	if zone != "" {
		result += "%" + zone
	}
	return result
}

func connOf(caller values.BalValue) *callerConn {
	conn, _ := caller.(*values.Object).Get("$conn")
	return conn.(*callerConn)
}

// respond sends resp unless a response was already sent; a 100 Continue goes out as an
// interim response and leaves the exchange open. A response that cannot be sent as given is
// replaced by a 500, and respond returns the reason.
func (c *callerConn) respond(ctx *extern.Context, resp *values.Object) values.BalValue {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed() {
		return httpError(ctx, genericListenerError, illegalRespond, nil)
	}
	if statusCode, _ := resp.Get("statusCode"); statusCode == int64(http.StatusContinue) {
		// net/http answers "Expect: 100-continue" itself when a bound parameter reads the body.
		if !c.state.has(stateBodyBound) || !strings.EqualFold(c.r.Header.Get("Expect"), "100-continue") {
			c.w.WriteHeader(http.StatusContinue)
		}
		return nil
	}
	c.state.set(stateResponded)
	err := writeResponse(c.rt, ctx, c.w, c.r, resp)
	c.markSent()
	if err != nil {
		return httpError(ctx, genericListenerError, err.Error(), nil)
	}
	return nil
}

// finish answers for a resource that has returned result. A resource that never responded
// gets a 500 with no body, or its returned error's response.
func (c *callerConn) finish(ctx *extern.Context, result values.BalValue) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.set(stateFinished)
	if c.state.has(stateResponded) {
		if _, isError := result.(*values.Error); isError {
			_, _ = c.rt.Platform().IO.Stderr([]byte("error: " + illegalReturn + "\nwarning: [ballerina/http] " + illegalReturn + "\n"))
		}
		return
	}
	c.state.set(stateResponded)
	if result == nil {
		c.w.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeResult(c.rt, ctx, c.types, c.w, c.r, c.accessor, result)
}

// isClosed reports whether no response can be sent any more.
func (c *callerConn) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed()
}

func nativeRespond(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return connOf(args[0]).respond(ctx, args[1].(*values.Object)), nil
}

func nativeRespondError(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	conn := connOf(args[0])
	// Checked first so an error that cannot be sent is not logged as handled.
	if conn.isClosed() {
		return httpError(ctx, genericListenerError, illegalRespond, nil), nil
	}
	resp, err := errorResponse(ctx, args[1].(*values.Error), conn.r)
	if err != nil {
		return nil, err
	}
	return conn.respond(ctx, resp), nil
}

func nativeGetRemoteHostName(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	conn := connOf(args[0])
	return remoteHostName(conn.rt, conn.r), nil
}

func registerCallerExterns(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "nativeRespond", nativeRespond)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "nativeRespondError", nativeRespondError)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "nativeGetRemoteHostName", nativeGetRemoteHostName)
}

// remoteHostName returns the first name a reverse lookup of the remote IP gives, or the IP
// itself when it has none.
func remoteHostName(rt *runtime.Runtime, r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if lookup := rt.Platform().Net.LookupAddr; lookup != nil {
		if names, err := lookup(context.Background(), host); err == nil && len(names) > 0 {
			return strings.TrimSuffix(names[0], ".")
		}
	}
	return javaHostAddress(host)
}
