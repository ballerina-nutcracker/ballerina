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
	"net"
	"strconv"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "udp"
)

// maxDatagramSize bounds a single outbound UDP fragment (jBallerina's DATAGRAM_DATA_SIZE).
const maxDatagramSize = 8192

// readBufferSize fits the largest UDP payload over IPv6 (65527 bytes) so reads never truncate.
const readBufferSize = 65535

type udpTypes struct {
	byteArrTy  semtypes.SemType
	datagramTy semtypes.SemType
}

func udpError(msg string) *values.Error {
	return values.NewErrorWithMessage(msg)
}

// resolveUDPAddr uses JoinHostPort so IPv6 literals are bracketed.
func resolveUDPAddr(rt *runtime.Runtime, host string, port int64) (net.Addr, error) {
	return rt.Platform().Net.ResolvePacketAddr("udp", net.JoinHostPort(host, strconv.FormatInt(port, 10)))
}

func hostPortOf(addr net.Addr) (string, int64, error) {
	host, portStr, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.ParseInt(portStr, 10, 64)
	return host, port, err
}

func listToBytes(list *values.List) []byte {
	b := make([]byte, list.Len())
	for i := 0; i < list.Len(); i++ {
		b[i] = byte(list.Get(i).(int64))
	}
	return b
}

func bytesToList(types udpTypes, ctx *extern.Context, data []byte, readonly bool) *values.List {
	vals := make([]values.BalValue, len(data))
	for i, b := range data {
		vals[i] = int64(b)
	}
	atomic := semtypes.ToListAtomicType(ctx.TypeEnv(), types.byteArrTy)
	return values.NewList(types.byteArrTy, atomic, readonly, nil, len(vals), vals)
}

func newDatagram(types udpTypes, ctx *extern.Context, remoteHost string, remotePort int64, data []byte, readonly bool) *values.Map {
	atomic := semtypes.ToMappingAtomicType(ctx.TypeCtx(), types.datagramTy)
	return values.NewMap(types.datagramTy, atomic, readonly, []values.MapEntry{
		{Key: "remoteHost", Value: remoteHost},
		{Key: "remotePort", Value: remotePort},
		{Key: "data", Value: bytesToList(types, ctx, data, readonly)},
	})
}

func initUdpModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	byteArrBld := semtypes.NewListDefinition()
	byteArrTy := byteArrBld.Define(env, nil, semtypes.ListRest(semtypes.Byte))

	datagramBld := semtypes.NewMappingDefinition()
	datagramTy := datagramBld.Define(env, []semtypes.Field{
		semtypes.FieldFrom("remoteHost", semtypes.String, false, false),
		semtypes.FieldFrom("remotePort", semtypes.Int, false, false),
		semtypes.FieldFrom("data", byteArrTy, false, false),
	}, semtypes.Never)

	types := udpTypes{byteArrTy: byteArrTy, datagramTy: datagramTy}

	registerClientFunctions(rt, types)
	registerConnectClientFunctions(rt, types)
	registerListenerFunctions(rt, types)
}

func init() {
	runtime.RegisterModuleInitializer(initUdpModule)
}
