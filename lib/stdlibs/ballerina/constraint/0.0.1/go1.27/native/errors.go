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
	"slices"
	"strings"
	"unicode/utf16"

	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	unexpectedErrorMessage     = "Unexpected error found due to typedesc and value mismatch."
	typeConversionErrorMessage = "Type conversion failed due to typedesc and value mismatch."
)

// newModuleError calls a helper declared in constraint_errors.bal because native code cannot build values of a
// distinct error type directly.
func newModuleError(ctx *extern.Context, helper string, args ...values.BalValue) (values.BalValue, error) {
	handle, ok := ctx.LookupFunction(orgName, moduleName, helper)
	if !ok {
		return nil, &invalidConstraintError{"constraint: internal helper function " + helper + " not found"}
	}
	return ctx.InvokeFunction(handle, args)
}

func validationError(ctx *extern.Context, failures []failure) (values.BalValue, error) {
	var withMessage, withoutMessage, allPaths []string
	for _, f := range failures {
		allPaths = append(allPaths, f.pathWithConstraint())
		if f.hasMessage {
			withMessage = append(withMessage, f.message)
		} else {
			withoutMessage = append(withoutMessage, f.pathWithConstraint())
		}
	}
	if len(withMessage) == 0 {
		return newModuleError(ctx, "newValidationError", defaultErrorMessage(allPaths), nil)
	}
	cause, err := newModuleError(ctx, "newError", defaultErrorMessage(allPaths))
	if err != nil {
		return nil, err
	}
	if len(withoutMessage) > 0 {
		withMessage = append(withMessage, defaultErrorMessage(withoutMessage))
	}
	return newModuleError(ctx, "newValidationError", joinMessages(withMessage), cause)
}

// defaultErrorMessage sorts by UTF-16 code units, the order Java's String comparison gives jBallerina.
func defaultErrorMessage(failed []string) string {
	sorted := slices.Clone(failed)
	slices.SortFunc(sorted, func(a, b string) int {
		return slices.Compare(utf16.Encode([]rune(a)), utf16.Encode([]rune(b)))
	})
	return "Validation failed for '" + strings.Join(sorted, "','") + "' constraint(s)."
}

func joinMessages(messages []string) string {
	var b strings.Builder
	for i, message := range messages {
		b.WriteString(trimMessage(message))
		switch {
		case i < len(messages)-2:
			b.WriteString(", ")
		case i == len(messages)-2:
			b.WriteString(" and ")
		}
	}
	b.WriteString(".")
	return b.String()
}

func trimMessage(message string) string {
	message = strings.TrimFunc(message, func(r rune) bool { return r <= ' ' })
	return strings.TrimSuffix(message, ".")
}
