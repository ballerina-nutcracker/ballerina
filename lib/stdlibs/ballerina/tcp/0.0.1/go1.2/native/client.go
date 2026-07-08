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
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"ballerina-lang-go/bir"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

const defaultTcpTimeout = 300 * time.Second

// readBufferSize bounds a single readBytes()/onBytes() read. There is no
// framing in this protocol (see README parity notes), so this is purely a
// buffer allocation size, not a message-size limit.
const readBufferSize = 32 * 1024

func connOf(self *values.Object) net.Conn {
	v, ok := self.Get("$conn")
	if !ok {
		return nil
	}
	conn, _ := v.(net.Conn)
	return conn
}

func classifyReadErr(err error) *values.Error {
	if errors.Is(err, net.ErrClosed) {
		return tcpError("Socket connection already closed.")
	}
	if errors.Is(err, io.EOF) {
		return tcpError("Connection closed by the server.")
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return tcpError("Read timed out")
	}
	return tcpError(err.Error())
}

func classifyWriteErr(err error) *values.Error {
	if errors.Is(err, net.ErrClosed) {
		return tcpError("Socket connection already closed.")
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return tcpError("Write timed out")
	}
	return tcpError("Failed to write data: " + err.Error())
}

func listToBytes(list *values.List) []byte {
	b := make([]byte, list.Len())
	for i := 0; i < list.Len(); i++ {
		b[i] = byte(list.Get(i).(int64))
	}
	return b
}

func bytesToList(types tcpTypes, ctx *extern.Context, data []byte) *values.List {
	vals := make([]values.BalValue, len(data))
	for i, b := range data {
		vals[i] = int64(b)
	}
	atomic := semtypes.ToListAtomicType(ctx.TypeCtx, types.byteArrTy)
	return values.NewList(types.byteArrTy, atomic, true, nil, len(vals), vals)
}

func registerClientFunctions(rt *runtime.Runtime, types tcpTypes) {
	clientClassDef := &bir.BIRClassDef{
		Name:      "Client",
		LookupKey: "ballerina/tcp:Client",
		VTable: map[string]*bir.BIRFunction{
			"init":               {FunctionLookupKey: "ballerina/tcp:Client.init"},
			"initTcpConnection":  {FunctionLookupKey: "ballerina/tcp:Client.initTcpConnection"},
			"$remote$writeBytes": {FunctionLookupKey: "ballerina/tcp:Client.$remote$writeBytes"},
			"$remote$readBytes":  {FunctionLookupKey: "ballerina/tcp:Client.$remote$readBytes"},
			"$remote$close":      {FunctionLookupKey: "ballerina/tcp:Client.$remote$close"},
		},
	}
	runtime.RegisterExternClassDef(rt, clientClassDef)

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.initTcpConnection",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			remoteHost := args[1].(string)
			remotePort := args[2].(int64)
			config := args[3].(*values.Map)

			localHost := ""
			if v, ok := config.Get("localHost"); ok {
				localHost, _ = v.(string)
			}
			timeout := defaultTcpTimeout
			if d, ok := decimalArg(config, "timeout"); ok {
				timeout = decimalToDuration(d)
			}
			writeTimeout := defaultTcpTimeout
			if d, ok := decimalArg(config, "writeTimeout"); ok {
				writeTimeout = decimalToDuration(d)
			}
			var secureSocket *values.Map
			if v, ok := config.Get("secureSocket"); ok {
				secureSocket, _ = v.(*values.Map)
			}
			tlsCfg, tlsErr := buildClientTLSConfig(rt, secureSocket)
			if tlsErr != nil {
				return tlsErr, nil
			}

			localAddr := ""
			if localHost != "" {
				localAddr = net.JoinHostPort(localHost, "0")
			}
			address := fmt.Sprintf("%s:%d", remoteHost, remotePort)
			dialCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			conn, err := rt.Platform().Net.Dial(dialCtx, "tcp", address, localAddr, tlsCfg)
			if err != nil {
				return tcpError("Unable to connect with remote host: " + err.Error()), nil
			}
			self.Put("$conn", conn)
			self.Put("$timeout", timeout)
			self.Put("$writeTimeout", writeTimeout)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$writeBytes",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			data := listToBytes(args[1].(*values.List))
			conn := connOf(self)
			if conn == nil {
				return tcpError("Socket connection already closed."), nil
			}
			writeTimeout, _ := self.Get("$writeTimeout")
			_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout.(time.Duration)))
			if _, err := conn.Write(data); err != nil {
				return classifyWriteErr(err), nil
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$readBytes",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			conn := connOf(self)
			if conn == nil {
				return tcpError("Socket connection already closed."), nil
			}
			timeout, _ := self.Get("$timeout")
			_ = conn.SetReadDeadline(time.Now().Add(timeout.(time.Duration)))
			buf := make([]byte, readBufferSize)
			n, err := conn.Read(buf)
			if err != nil {
				return classifyReadErr(err), nil
			}
			return bytesToList(types, ctx, buf[:n]), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$close",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			conn := connOf(self)
			if conn == nil {
				return nil, nil
			}
			if err := conn.Close(); err != nil {
				return tcpError("Unable to close the TCP client: " + err.Error()), nil
			}
			return nil, nil
		})
}
