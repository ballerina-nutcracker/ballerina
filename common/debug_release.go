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

//go:build !debug

package common

import "io"

const (
	DUMP_TOKENS uint16 = 1 << iota
	DUMP_ST
	DEBUG_ERROR_RECOVERY
)

func InitDebug(_ uint16, _ io.Writer)          {}
func DebugFlags() uint16                       { return 0 }
func DebugWriter() io.Writer                   { return io.Discard }
func DebugEnabled(_ uint16) bool               { return false }
func DebugWriteLazy(_ uint16, _ func() string) {}
func WithSuppressedDebug(fn func())            { fn() }
