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

package frame

import (
	"sync"

	"ballerina-lang-go/values"
)

var (
	localsPool8   = sync.Pool{New: func() any { return new([8]values.BalValue) }}
	localsPool16  = sync.Pool{New: func() any { return new([16]values.BalValue) }}
	localsPool32  = sync.Pool{New: func() any { return new([32]values.BalValue) }}
	localsPool64  = sync.Pool{New: func() any { return new([64]values.BalValue) }}
	localsPool128 = sync.Pool{New: func() any { return new([128]values.BalValue) }}
)

type Frame struct {
	locals      []values.BalValue
	functionKey string
	parent      *Frame
	escaped     bool
}

// New create a frame for holding numLocals. The backing memory may be reused from already
// freed frames there for locals may have garbage. Caller must either clean the memory or more
// likely overwrite it.
func New(numLocals int, parent *Frame) *Frame {
	return &Frame{locals: newLocals(numLocals), parent: parent}
}

func newLocals(numLocals int) []values.BalValue {
	switch {
	case numLocals <= 0:
		return make([]values.BalValue, numLocals)
	case numLocals <= 8:
		return localsPool8.Get().(*[8]values.BalValue)[:numLocals]
	case numLocals <= 16:
		return localsPool16.Get().(*[16]values.BalValue)[:numLocals]
	case numLocals <= 32:
		return localsPool32.Get().(*[32]values.BalValue)[:numLocals]
	case numLocals <= 64:
		return localsPool64.Get().(*[64]values.BalValue)[:numLocals]
	case numLocals <= 128:
		return localsPool128.Get().(*[128]values.BalValue)[:numLocals]
	default:
		return make([]values.BalValue, numLocals)
	}
}

func (f *Frame) Free() {
	if f.escaped {
		// Calling free here is deceptive given that this frame has been captured
		// and can't be freed.
		return
	}

	locals := f.locals
	switch cap(locals) {
	case 8:
		localsPool8.Put((*[8]values.BalValue)(locals[:8]))
	case 16:
		localsPool16.Put((*[16]values.BalValue)(locals[:16]))
	case 32:
		localsPool32.Put((*[32]values.BalValue)(locals[:32]))
	case 64:
		localsPool64.Put((*[64]values.BalValue)(locals[:64]))
	case 128:
		localsPool128.Put((*[128]values.BalValue)(locals[:128]))
	}
}

func (f *Frame) MarkEscaped() {
	for f != nil {
		f.escaped = true
		f = f.parent
	}
}

func (f *Frame) Parent() *Frame {
	return f.parent
}

func (f *Frame) FunctionKey() string {
	return f.functionKey
}

func (f *Frame) SetFunctionKey(functionKey string) {
	f.functionKey = functionKey
}

func (f *Frame) Local(idx int) values.BalValue {
	return f.locals[idx]
}

func (f *Frame) SetLocal(idx int, value values.BalValue) {
	f.locals[idx] = value
}
