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
	"crypto/rand"
	"fmt"
	"regexp"
	"strconv"

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "uuid"

	// gregorianOffset is the number of 100-nanosecond intervals between
	// Oct 15, 1582 (UUID epoch) and Jan 1, 1970 (Unix epoch).
	gregorianOffset = uint64(122192928000000000)
)

var uuidRegexp = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func initUUIDModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externCreateType1AsString",
		func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
			now := rt.Platform().Time.Now()

			// Convert current time to 100-nanosecond intervals since UUID epoch.
			timestamp := uint64(now.UnixNano()/100) + gregorianOffset

			timeLow := uint32(timestamp & 0xffffffff)
			timeMid := uint16((timestamp >> 32) & 0xffff)
			timeHi := uint16((timestamp>>48)&0x0fff) | 0x1000 // version 1

			// 2-byte random clock sequence.
			var clockBytes [2]byte
			if _, err := rand.Read(clockBytes[:]); err != nil {
				return nil, fmt.Errorf("uuid v1: failed to generate clock sequence: %w", err)
			}
			clockSeq := (uint16(clockBytes[0])<<8 | uint16(clockBytes[1])) & 0x3fff
			clockSeq |= 0x8000 // variant bits (10xxxxxx)

			// 6-byte random node ID (no MAC address access for portability).
			var node [6]byte
			if _, err := rand.Read(node[:]); err != nil {
				return nil, fmt.Errorf("uuid v1: failed to generate node: %w", err)
			}
			nodeInt := uint64(node[0])<<40 | uint64(node[1])<<32 | uint64(node[2])<<24 |
				uint64(node[3])<<16 | uint64(node[4])<<8 | uint64(node[5])

			return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
				timeLow, timeMid, timeHi, clockSeq, nodeInt), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externCreateType4AsString",
		func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
			var uuid [16]byte
			if _, err := rand.Read(uuid[:]); err != nil {
				return nil, fmt.Errorf("uuid v4: failed to generate random bytes: %w", err)
			}
			uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
			uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant bits (10xxxxxx)

			return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
				uuid[0:4],
				uuid[4:6],
				uuid[6:8],
				uuid[8:10],
				uuid[10:16]), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externValidate", validateExtern)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externParseHexUint", parseHexUintExtern)
}

// validateExtern's argument is declared `string` in uuid.bal, so the
// Ballerina type checker guarantees args[0] is always a string.
func validateExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return uuidRegexp.MatchString(args[0].(string)), nil
}

// parseHexUintExtern's argument is declared `string` in uuid.bal, so the
// Ballerina type checker guarantees args[0] is always a string.
func parseHexUintExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	s := args[0].(string)
	n, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		return values.NewErrorWithMessage("invalid hex string: \"" + s + "\""), nil
	}
	return int64(n), nil
}

func init() {
	runtime.RegisterModuleInitializer(initUUIDModule)
}
