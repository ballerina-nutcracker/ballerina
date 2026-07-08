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
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "udp"
)

// maxDatagramSize bounds a single outbound UDP fragment (matches jBallerina's
// DATAGRAM_DATA_SIZE) and the buffer used for a single inbound read (the
// practical max UDP payload over IPv4, well above any realistic datagram).
const maxDatagramSize = 8192

const readBufferSize = 65507

// udpTypes holds the SemTypes shared by every native function in this
// module. Unlike the generic-open-map convention used by ldap/crypto/file
// (all their record fields share one type), Datagram mixes string/int/byte[]
// fields, so it needs a hand-built closed record shape instead of a generic
// open map — see FieldFrom below.
type udpTypes struct {
	byteArrTy  semtypes.SemType
	datagramTy semtypes.SemType
}

func udpError(msg string) *values.Error {
	return values.NewErrorWithMessage(msg)
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
	atomic := semtypes.ToListAtomicType(ctx.TypeCtx, types.byteArrTy)
	return values.NewList(types.byteArrTy, atomic, readonly, nil, len(vals), vals)
}

func newDatagram(types udpTypes, ctx *extern.Context, remoteHost string, remotePort int64, data []byte, readonly bool) *values.Map {
	atomic := semtypes.ToMappingAtomicType(ctx.TypeCtx, types.datagramTy)
	return values.NewMap(types.datagramTy, atomic, readonly, []values.MapEntry{
		{Key: "remoteHost", Value: remoteHost},
		{Key: "remotePort", Value: remotePort},
		{Key: "data", Value: bytesToList(types, ctx, data, readonly)},
	})
}

func initUdpModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	byteArrBld := semtypes.NewListDefinition()
	byteArrTy := byteArrBld.DefineListTypeWrappedWithEnvSemType(env, semtypes.BYTE)

	datagramBld := semtypes.NewMappingDefinition()
	datagramTy := datagramBld.DefineMappingTypeWrapped(env, []semtypes.Field{
		semtypes.FieldFrom("remoteHost", semtypes.STRING, false, false),
		semtypes.FieldFrom("remotePort", semtypes.INT, false, false),
		semtypes.FieldFrom("data", byteArrTy, false, false),
	}, semtypes.NEVER)

	types := udpTypes{byteArrTy: byteArrTy, datagramTy: datagramTy}

	registerClientFunctions(rt, types)
	registerConnectClientFunctions(rt, types)
	registerListenerFunctions(rt, types)
}

func init() {
	runtime.RegisterModuleInitializer(initUdpModule)
}
