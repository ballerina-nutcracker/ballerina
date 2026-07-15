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

package values

import "ballerina-lang-go/semtypes"

// ToByteSlice converts a Ballerina byte[] (List of int64 in 0-255) to a Go []byte.
func (l *List) ToByteSlice() []byte {
	b := make([]byte, l.Len())
	for i := 0; i < l.Len(); i++ {
		b[i] = byte(l.Get(i).(int64))
	}
	return b
}

// ByteSliceToList converts a Go []byte to a Ballerina byte[] (List).
func ByteSliceToList(byteArrTy semtypes.SemType, tc semtypes.Context, data []byte) *List {
	items := make([]BalValue, len(data))
	for i, b := range data {
		items[i] = int64(b)
	}
	return NewList(byteArrTy, semtypes.ToListAtomicType(tc, byteArrTy), false, nil, 0, items)
}
