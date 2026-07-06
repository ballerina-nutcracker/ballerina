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

package exec

import (
	"fmt"
	"path/filepath"
	"strings"

	"ballerina-lang-go/bir"
	"ballerina-lang-go/values"
)

func getFormattedError(cs *callStack, r any) error {
	message := panicMessage(r)
	stack := formatCallStack(cs)
	return fmt.Errorf("%s", formatRuntimePanic(message, stack))
}

func panicMessage(r any) string {
	switch v := r.(type) {
	case *values.Error:
		return v.Message
	case error:
		return v.Error()
	default:
		return fmt.Sprintf("%v", r)
	}
}

func formatCallStack(cs *callStack) []string {
	entries := cs.Entries()
	const maxFrames = 32
	out := make([]string, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		if len(out) >= maxFrames {
			out = append(out, "...")
			break
		}
		entry := entries[i]
		loc := entry.location
		if bir.IsLocationEmpty(loc) {
			out = append(out, fmt.Sprintf("%s(unknown)", prettyFunctionName(entry.frame.FunctionKey())))
			continue
		}
		file := filepath.Base(loc.FilePath())
		line := loc.StartLine() + 1
		out = append(out, fmt.Sprintf("%s(%s:%d)", prettyFunctionName(entry.frame.FunctionKey()), file, line))
	}
	return out
}

func formatRuntimePanic(message string, stack []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "error: %s\n", message)
	if len(stack) > 0 {
		fmt.Fprintf(&b, "        at %s\n", stack[0])
		for _, line := range stack[1:] {
			fmt.Fprintf(&b, "           %s\n", line)
		}
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func prettyFunctionName(functionKey string) string {
	// For anonymous single-file modules, drop the module prefix and keep only the function name.
	// Example: "$anon/stack-overflow:main" -> "main"
	if strings.HasPrefix(functionKey, "$anon/") {
		if idx := strings.LastIndex(functionKey, ":"); idx != -1 && idx+1 < len(functionKey) {
			return functionKey[idx+1:]
		}
	}
	return functionKey
}
