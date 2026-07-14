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
	"net"
	"strconv"
	"time"

	"ballerina-lang-go/bir"
	"ballerina-lang-go/decimal"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
)

const defaultUdpTimeout = 300 * time.Second

func decimalArg(m *values.Map, key string) (*decimal.Decimal, bool) {
	v, ok := m.Get(key)
	if !ok {
		return nil, false
	}
	d, ok := v.(*decimal.Decimal)
	return d, ok
}

func decimalToDuration(d *decimal.Decimal) time.Duration {
	return time.Duration(d.Float64() * float64(time.Second))
}

func readTimeoutOf(config *values.Map) time.Duration {
	if d, ok := decimalArg(config, "timeout"); ok {
		return decimalToDuration(d)
	}
	return defaultUdpTimeout
}

func localHostOf(config *values.Map) string {
	if v, ok := config.Get("localHost"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func classifyPacketErr(err error, timeoutMsg string) *values.Error {
	if errors.Is(err, net.ErrClosed) {
		return udpError("Socket connection already closed.")
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return udpError(timeoutMsg)
	}
	return udpError(err.Error())
}

func fragmentBytes(data []byte) [][]byte {
	if len(data) == 0 {
		return [][]byte{data}
	}
	fragments := make([][]byte, 0, (len(data)+maxDatagramSize-1)/maxDatagramSize)
	for len(data) > 0 {
		n := maxDatagramSize
		if n > len(data) {
			n = len(data)
		}
		fragments = append(fragments, data[:n])
		data = data[n:]
	}
	return fragments
}

func datagramFields(datagram *values.Map) (string, int64, []byte) {
	host, _ := datagram.Get("remoteHost")
	port, _ := datagram.Get("remotePort")
	data, _ := datagram.Get("data")
	return host.(string), port.(int64), listToBytes(data.(*values.List))
}

func registerClientFunctions(rt *runtime.Runtime, types udpTypes) {
	clientClassDef := &bir.BIRClassDef{
		Name:      "Client",
		LookupKey: "ballerina/udp:Client",
		VTable: map[string]*bir.BIRFunction{
			"init":                 {FunctionLookupKey: "ballerina/udp:Client.init"},
			"initClient":           {FunctionLookupKey: "ballerina/udp:Client.initClient"},
			"$remote$sendDatagram": {FunctionLookupKey: "ballerina/udp:Client.$remote$sendDatagram"},
			"$remote$receiveDatagram": {
				FunctionLookupKey: "ballerina/udp:Client.$remote$receiveDatagram",
			},
			"$remote$close": {FunctionLookupKey: "ballerina/udp:Client.$remote$close"},
		},
	}
	runtime.RegisterExternClassDef(rt, clientClassDef)

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.initClient",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			config := args[1].(*values.Map)

			address := net.JoinHostPort(localHostOf(config), "0")
			pc, err := rt.Platform().Net.ListenPacket("udp", address)
			if err != nil {
				return udpError("Error initializing UDP Client: " + err.Error()), nil
			}
			self.Put("$pc", pc)
			self.Put("$timeout", readTimeoutOf(config))
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$sendDatagram",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			host, port, data := datagramFields(args[1].(*values.Map))
			pc := packetConnOf(self)
			if pc == nil {
				return udpError("Socket connection already closed."), nil
			}
			addr, err := resolveUDPAddr(host, port)
			if err != nil {
				return udpError("Failed to send data: " + err.Error()), nil
			}
			if werr := writeFragments(pc, addr, data); werr != nil {
				return udpError("Failed to send data: " + werr.Error()), nil
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$receiveDatagram",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			pc := packetConnOf(self)
			if pc == nil {
				return udpError("Socket connection already closed."), nil
			}
			timeout, _ := self.Get("$timeout")
			_ = pc.SetReadDeadline(time.Now().Add(timeout.(time.Duration)))
			buf := make([]byte, readBufferSize)
			n, addr, err := pc.ReadFrom(buf)
			if err != nil {
				return classifyPacketErr(err, "Read timed out"), nil
			}
			host, portStr, _ := net.SplitHostPort(addr.String())
			port, _ := strconv.ParseInt(portStr, 10, 64)
			return newDatagram(types, ctx, host, port, buf[:n], true), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$close",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			pc := packetConnOf(self)
			if pc == nil {
				return nil, nil
			}
			if err := pc.Close(); err != nil {
				return udpError("Unable to close the UDP client: " + err.Error()), nil
			}
			return nil, nil
		})
}

func registerConnectClientFunctions(rt *runtime.Runtime, types udpTypes) {
	connectClientClassDef := &bir.BIRClassDef{
		Name:      "ConnectClient",
		LookupKey: "ballerina/udp:ConnectClient",
		VTable: map[string]*bir.BIRFunction{
			"init":               {FunctionLookupKey: "ballerina/udp:ConnectClient.init"},
			"initConnectClient":  {FunctionLookupKey: "ballerina/udp:ConnectClient.initConnectClient"},
			"$remote$writeBytes": {FunctionLookupKey: "ballerina/udp:ConnectClient.$remote$writeBytes"},
			"$remote$readBytes":  {FunctionLookupKey: "ballerina/udp:ConnectClient.$remote$readBytes"},
			"$remote$close":      {FunctionLookupKey: "ballerina/udp:ConnectClient.$remote$close"},
		},
	}
	runtime.RegisterExternClassDef(rt, connectClientClassDef)

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ConnectClient.initConnectClient",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			remoteHost := args[1].(string)
			remotePort := args[2].(int64)
			config := args[3].(*values.Map)

			localAddr := ""
			if localHost := localHostOf(config); localHost != "" {
				localAddr = net.JoinHostPort(localHost, "0")
			}
			address := net.JoinHostPort(remoteHost, strconv.FormatInt(remotePort, 10))
			dialCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			conn, err := rt.Platform().Net.DialPacket(dialCtx, "udp", address, localAddr)
			if err != nil {
				return udpError("Can't connect to remote host: " + err.Error()), nil
			}
			self.Put("$conn", conn)
			self.Put("$timeout", readTimeoutOf(config))
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ConnectClient.$remote$writeBytes",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			data := listToBytes(args[1].(*values.List))
			conn := connOf(self)
			if conn == nil {
				return udpError("Socket connection already closed."), nil
			}
			for _, fragment := range fragmentBytes(data) {
				if _, err := conn.Write(fragment); err != nil {
					return udpError("Failed to send data: " + err.Error()), nil
				}
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ConnectClient.$remote$readBytes",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			conn := connOf(self)
			if conn == nil {
				return udpError("Socket connection already closed."), nil
			}
			timeout, _ := self.Get("$timeout")
			_ = conn.SetReadDeadline(time.Now().Add(timeout.(time.Duration)))
			buf := make([]byte, readBufferSize)
			n, err := conn.Read(buf)
			if err != nil {
				return classifyPacketErr(err, "Read timed out"), nil
			}
			return bytesToList(types, ctx, buf[:n], true), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ConnectClient.$remote$close",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			conn := connOf(self)
			if conn == nil {
				return nil, nil
			}
			if err := conn.Close(); err != nil {
				return udpError("Unable to close the UDP client: " + err.Error()), nil
			}
			return nil, nil
		})
}

func packetConnOf(self *values.Object) net.PacketConn {
	v, ok := self.Get("$pc")
	if !ok {
		return nil
	}
	pc, _ := v.(net.PacketConn)
	return pc
}

func connOf(self *values.Object) net.Conn {
	v, ok := self.Get("$conn")
	if !ok {
		return nil
	}
	conn, _ := v.(net.Conn)
	return conn
}

func writeFragments(pc net.PacketConn, addr net.Addr, data []byte) error {
	for _, fragment := range fragmentBytes(data) {
		if _, err := pc.WriteTo(fragment, addr); err != nil {
			return err
		}
	}
	return nil
}
