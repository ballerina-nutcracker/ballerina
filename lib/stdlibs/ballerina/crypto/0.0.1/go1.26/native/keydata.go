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
	stdruntime "runtime"
	"sync"
	"weak"

	"ballerina/values"
)

// keyData associates opaque Go key material (an *rsa.PrivateKey, *rsa.PublicKey,
// *ecdsa.PrivateKey, or *ecdsa.PublicKey) with the *values.Map backing a
// crypto:PrivateKey/PublicKey record.
//
// Ballerina mapping values are pure data with no encapsulation: nothing stops
// a caller from constructing a same-shaped record without ever going through
// decode/generate, so every reader here must tolerate a miss (see the
// "Uninitialized ... key" errors in rsa.go) rather than assume the
// association always exists. Keying by a weak pointer keeps the association
// entirely internal to this module — the runtime and other modules never see
// it — and it self-cleans once the Map becomes unreachable, so it cannot leak
// memory for keys that are dropped by the program.
//
// A sync.Map fits this access pattern (each entry written once at decode time
// and read many times by sign/verify/encrypt/decrypt) so concurrent key
// lookups from different strands proceed without serializing on a single lock.
var keyData sync.Map // weak.Pointer[values.Map] -> any (the Go key)

func setKeyData(m *values.Map, data any) {
	wp := weak.Make(m)
	keyData.Store(wp, data)
	stdruntime.AddCleanup(m, func(p weak.Pointer[values.Map]) {
		keyData.Delete(p)
	}, wp)
}

func keyDataOf(m *values.Map) any {
	data, _ := keyData.Load(weak.Make(m))
	return data
}
