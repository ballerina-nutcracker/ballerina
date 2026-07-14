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
)

const (
	DUMP_TOKENS uint16 = 1 << iota
	DUMP_ST
	DEBUG_ERROR_RECOVERY
)

var debugFlags uint16
var debugWriter io.Writer = os.Stderr

func InitDebug(flags uint16, w io.Writer) {
	debugFlags = flags
	if w != nil {
		debugWriter = w
	}
}

func DebugFlags() uint16     { return debugFlags }
func DebugWriter() io.Writer { return debugWriter }

func DebugEnabled(flag uint16) bool {
	return debugFlags&flag != 0
}

func DebugWriteLazy(flag uint16, msgFn func() string) {
	if debugFlags&flag != 0 {
		fmt.Fprintf(debugWriter, "%s\n", msgFn())
	}
}

func WithSuppressedDebug(fn func()) {
	saved := debugFlags
	debugFlags = 0
	defer func() { debugFlags = saved }()
	fn()
}
