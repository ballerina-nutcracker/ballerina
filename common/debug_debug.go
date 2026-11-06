// Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
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

//go:build debug

package common

import (
	"fmt"
	"io"
	"os"
	"sync"
)

const (
	DUMP_TOKENS uint16 = 1 << iota
	DUMP_ST
	DEBUG_ERROR_RECOVERY
)

var debugFlags uint16
var debugWriter io.Writer = os.Stderr
var debugMu sync.RWMutex
var debugWriteMu sync.Mutex
var debugSuppressionDepth int

func InitDebug(flags uint16, w io.Writer) {
	debugMu.Lock()
	defer debugMu.Unlock()
	debugFlags = flags
	if w != nil {
		debugWriter = w
	}
}

func DebugWriteLazy(flag uint16, msgFn func() string) {
	debugMu.RLock()
	enabled := debugSuppressionDepth == 0 && debugFlags&flag != 0
	writer := debugWriter
	debugMu.RUnlock()
	if enabled {
		message := msgFn()
		debugWriteMu.Lock()
		defer debugWriteMu.Unlock()
		_, _ = fmt.Fprintf(writer, "%s\n", message)
	}
}

func WithSuppressedDebug(fn func()) {
	debugMu.Lock()
	debugSuppressionDepth++
	debugMu.Unlock()
	defer func() {
		debugMu.Lock()
		debugSuppressionDepth--
		debugMu.Unlock()
	}()
	fn()
}
