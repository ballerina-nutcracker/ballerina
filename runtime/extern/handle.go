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

package extern

import "ballerina/values"

// MethodHandle is an opaque reference to a resolved method on a Ballerina
// object. Obtain one from Context.LookupObjectMethod,
// Context.LookupRemoteMethod, or Context.LookupResourceMethod and pass it
// to Context.InvokeMethod.
type MethodHandle struct {
	impl any
}

// FunctionHandle is an opaque reference to a function,
// returned by Runtime.LookupFunction or Context.LookupFunction
type FunctionHandle struct {
	Fn any
}

// DispatchHandles carry the runtime's method-resolution and invocation
// implementations. They are installed once by InitEnv and used by the
// Context.Lookup*/InvokeMethod/StartMethod methods. Lookup hooks return the
// resolved payload along with a found bool; Context methods forward both.
type DispatchHandles struct {
	LookupObject   func(*Context, *values.Object, string) (any, bool)
	LookupRemote   func(*Context, *values.Object, string) (any, bool)
	LookupResource func(*Context, *values.Object, string, []values.BalValue) (any, bool) // resourceMethodName, path
	LookupFunction func(*Context, string, string, string) (any, bool)                    // org, module, name
	Invoke         func(*Context, any, []values.BalValue) (values.BalValue, error)
	Start          func(*Context, any, []values.BalValue) (<-chan values.BalValue, error)
}

// LookupObjectMethod resolves a regular method on obj. The second return is
// false if obj has no such method. Remote methods are not resolved through
// this entry point.
func (c *Context) LookupObjectMethod(obj *values.Object, name string) (MethodHandle, bool) {
	impl, ok := c.Env.dispatch.LookupObject(c, obj, name)
	return MethodHandle{impl: impl}, ok
}

// LookupRemoteMethod resolves a remote method on obj. Pass the bare method
// name; the runtime applies the remote-method mangling internally. The
// second return is false if obj has no such remote method.
func (c *Context) LookupRemoteMethod(obj *values.Object, name string) (MethodHandle, bool) {
	impl, ok := c.Env.dispatch.LookupRemote(c, obj, name)
	return MethodHandle{impl: impl}, ok
}

// LookupResourceMethod resolves a resource method for given method name and path parameters.
// The second return is false if no candidate matches or if more than one
// candidate matches (ambiguous dispatch).
//
// path carries a value for every segment of the source-level resource
// access expression, including literal segments.
func (c *Context) LookupResourceMethod(obj *values.Object, resourceMethodName string, path []values.BalValue) (MethodHandle, bool) {
	impl, ok := c.Env.dispatch.LookupResource(c, obj, resourceMethodName, path)
	return MethodHandle{impl: impl}, ok
}

// InvokeMethod calls the method captured by h. For object and remote
// handles args is the full argument list including the receiver at
// index 0. For resource handles the receiver and path are already baked
// into the handle; args is only the user-supplied call arguments.
func (c *Context) InvokeMethod(h MethodHandle, args []values.BalValue) (values.BalValue, error) {
	return c.Env.dispatch.Invoke(c, h.impl, args)
}

// LookupFunction resolves a top-level BIR function by qualified name.
// The second return is false if no such function is registered.
func (c *Context) LookupFunction(org, module, name string) (FunctionHandle, bool) {
	impl, ok := c.Env.dispatch.LookupFunction(c, org, module, name)
	return FunctionHandle{Fn: impl}, ok
}

// InvokeFunction calls the function captured by h.
func (c *Context) InvokeFunction(h FunctionHandle, args []values.BalValue) (values.BalValue, error) {
	return c.Env.dispatch.Invoke(c, h.Fn, args)
}

// StartMethod is the non-blocking counterpart to InvokeMethod. It spawns a
// new strand to execute h and returns a buffered channel of capacity 1
// that will receive exactly one BalValue and then be closed.
//
// The returned error is reserved for synchronous failures to schedule the
// strand
// Asynchronous failures —  will be returned as the error value in the channel
func (c *Context) StartMethod(h MethodHandle, args []values.BalValue) (<-chan values.BalValue, error) {
	return c.Env.dispatch.Start(c, h.impl, args)
}
