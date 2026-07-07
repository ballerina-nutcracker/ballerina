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
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "crypto"
)

type cryptoTypes struct {
	byteArrTy semtypes.SemType
	keyMapTy  semtypes.SemType
	utcTy     semtypes.SemType
}

// cryptoError builds a Ballerina Error value with the given message.
func cryptoError(msg string) *values.Error {
	return values.NewErrorWithMessage(msg)
}

// listToBytes converts a Ballerina byte[] (values.List) to a Go []byte.
func listToBytes(list *values.List) []byte {
	b := make([]byte, list.Len())
	for i := range list.Len() {
		b[i] = byte(list.Get(i).(int64))
	}
	return b
}

// bytesToList converts a Go []byte to a Ballerina byte[] (values.List).
func bytesToList(byteArrTy semtypes.SemType, ctx *extern.Context, data []byte) *values.List {
	items := make([]values.BalValue, len(data))
	for i, b := range data {
		items[i] = int64(b)
	}
	return values.NewList(byteArrTy, semtypes.ToListAtomicType(ctx.TypeCtx, byteArrTy), false, nil, 0, items)
}

// mapString reads a string field from a Ballerina Map.
func mapString(m *values.Map, key string) string {
	v, _ := m.Get(key)
	s, _ := v.(string)
	return s
}
