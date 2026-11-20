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
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// representable tells whether the platform can name a file by this path, as Java's Paths.get does.
func representable(path string) bool {
	if strings.ContainsRune(path, 0) {
		return false
	}
	if filepath.Separator != '\\' {
		return true
	}
	for i, c := range path {
		if c < 0x20 || strings.ContainsRune(`<>"|?*`, c) || c == ':' && i != 1 {
			return false
		}
	}
	return true
}

func invalidPathError(ctx *extern.Context, path string) (values.BalValue, error) {
	return zipError(ctx, fileSystem, fmt.Sprintf("invalid path '%s': the platform cannot represent it", path))
}

// javaPathString is the path as Java's Path.toString gives it: separators that repeat or end the path are
// dropped, while `.` and `..` are kept.
func javaPathString(path string) string {
	sep := string(filepath.Separator)
	if sep != "/" {
		path = strings.ReplaceAll(path, "/", sep)
	}
	var b strings.Builder
	for i := 0; i < len(path); i++ {
		if path[i] == filepath.Separator && i > 0 && path[i-1] == filepath.Separator {
			continue
		}
		b.WriteByte(path[i])
	}
	result := b.String()
	if len(result) > 1 && strings.HasSuffix(result, sep) {
		result = result[:len(result)-1]
	}
	return result
}

func (e *zipExterns) realPath(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	path, _ := args[0].(string)
	if !representable(path) {
		return invalidPathError(ctx, path)
	}
	real, err := e.rt.Platform().FS.RealPath(path)
	if err != nil {
		return zipError(ctx, fileSystem, fmt.Sprintf("cannot resolve the path '%s'", path))
	}
	return real, nil
}

func (e *zipExterns) isSameFile(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	first, _ := args[0].(string)
	second, _ := args[1].(string)
	if !representable(first) {
		return invalidPathError(ctx, first)
	}
	if !representable(second) {
		return invalidPathError(ctx, second)
	}
	same, err := e.rt.Platform().FS.SameFile(javaPathString(first), javaPathString(second))
	if err != nil {
		return zipError(ctx, fileSystem, fmt.Sprintf("cannot compare the paths '%s' and '%s'", first, second))
	}
	return same, nil
}

// setModifiedTime never fails: a platform that cannot store the time leaves the file with the one it has.
func (e *zipExterns) setModifiedTime(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	path, _ := args[0].(string)
	seconds, _ := args[1].(int64)
	nanos, _ := args[2].(int64)
	if representable(path) {
		_ = e.rt.Platform().FS.Chtimes(path, time.Unix(seconds, nanos))
	}
	return nil, nil
}
