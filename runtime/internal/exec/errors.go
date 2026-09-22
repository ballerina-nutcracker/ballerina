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

	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// panicWithStack is used by futures to hold a panic so until it is safe to rethrow it.
// Currently futures only propagate panics from futures strand only at wait. I think we can
// do this at any yeild point. We can't panic arbiterary because of alternate wait. Alternate wait only
// completes abruptly if one future completes abtruptly while waiting; so after one completes successful
// subsequent panics must be silently dropped.
type panicWithStack struct {
	value any
	stack []string
}

func capturePanicStack(value any, stack []string) any {
	if _, ok := value.(*panicWithStack); ok {
		return value
	}
	return &panicWithStack{value: value, stack: append([]string(nil), stack...)}
}

func originalPanicValue(value any) any {
	if captured, ok := value.(*panicWithStack); ok {
		return captured.value
	}
	return value
}

func getFormattedError(cs *callStack, r any) error {
	stack := formatCallStack(cs)
	if captured, ok := r.(*panicWithStack); ok {
		r = captured.value
		stack = captured.stack
	}
	message := panicMessage(r)
	return fmt.Errorf("%s", formatRuntimePanic(message, stack))
}

// panicWithExternError raises a native-call failure as a *values.Error so
// that `trap` can recover it like any other Ballerina panic.
func panicWithExternError(err error) {
	panic(values.NewErrorWithMessage(err.Error()))
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

type stackFrame struct {
	name string
	loc  bir.Location
}

func formatCallStack(cs *callStack) []string {
	entries := cs.Entries()
	const maxFrames = 32
	frames := make([]stackFrame, 0, len(entries))
	var hiddenLocation bir.Location
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if isDesugaredFunction(entry.frame.FunctionKey()) {
			// Keep the innermost hidden location so the panic site is still reported.
			if bir.IsLocationEmpty(hiddenLocation) {
				hiddenLocation = entry.location
			}
			continue
		}
		name := prettyFunctionName(entry.frame.FunctionKey())
		if !bir.IsLocationEmpty(hiddenLocation) {
			// The panic happened in a hidden desugared frame that ran inside this
			// function, so that site is the innermost frame worth reporting.
			enclosed := enclosesSourceLine(entry.location, hiddenLocation)
			frames = append(frames, stackFrame{name: name, loc: hiddenLocation})
			hiddenLocation = bir.Location{}
			// If the panic site lies inside the statement this frame already
			// points at, both entries would describe the same construct.
			if enclosed {
				continue
			}
		}
		frames = append(frames, stackFrame{name: name, loc: entry.location})
	}

	out := make([]string, 0, len(frames))
	for _, frame := range frames {
		if len(out) >= maxFrames {
			out = append(out, "...")
			break
		}
		out = append(out, formatStackFrame(frame))
	}
	return out
}

func formatStackFrame(frame stackFrame) string {
	if bir.IsLocationEmpty(frame.loc) {
		return fmt.Sprintf("%s(unknown)", frame.name)
	}
	return fmt.Sprintf("%s(%s:%d)", frame.name, filepath.Base(frame.loc.FilePath()), frame.loc.StartLine()+1)
}

// enclosesSourceLine reports whether inner starts inside outer's line range.
func enclosesSourceLine(outer, inner bir.Location) bool {
	return outer.FilePath() == inner.FilePath() &&
		inner.StartLine() >= outer.StartLine() && inner.StartLine() <= outer.EndLine()
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

func isDesugaredFunction(functionKey string) bool {
	name := functionKey
	if idx := strings.LastIndex(name, ":"); idx != -1 {
		name = name[idx+1:]
	}
	return strings.HasPrefix(name, "$default$") ||
		strings.HasPrefix(name, "$anonFunc$")
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
