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

// Stream is the runtime representation of a stream<T, C> value.
//
// next/close are resolved against the backing object (the StreamImplementor
// passed to `new stream<T, C>(impl)`) once at construction time, so each
// element access avoids a virtual dispatch. close is nil when the
// implementor does not define a close method.
// If we are creating streams in native code and can provide more efficient implementations
// for next or close then they should be used instead
type Stream struct {
	Type  semtypes.SemType
	Next  func() BalValue
	Close func() BalValue
}

func NewStream(typ semtypes.SemType, next, close func() BalValue) *Stream {
	return &Stream{
		Type:  typ,
		Next:  next,
		Close: close,
	}
}
