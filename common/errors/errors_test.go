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

package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestDecorateWithStackTrace_WithBacktraceDisabled(t *testing.T) {
	t.Setenv("BAL_BACKTRACE", "")

	err := errors.New("test error")
	decorated := DecorateWithStackTrace(err)

	// When backtrace is disabled, should return the original error
	if decorated.Error() != "test error" {
		t.Errorf("Expected 'test error', got '%s'", decorated.Error())
	}

	// Should be the same error reference when backtrace is disabled
	if decorated != err {
		t.Error("Expected original error when backtrace is disabled")
	}
}

func TestDecorateWithStackTrace_WithBacktraceEnabled(t *testing.T) {
	// Enable backtrace
	t.Setenv("BAL_BACKTRACE", "true")

	err := errors.New("test error with stack")
	decorated := DecorateWithStackTrace(err)

	// Should not be the original error when backtrace is enabled
	if decorated == err {
		t.Fatal("Expected decorated error when backtrace is enabled")
	}

	// Error message should contain original error
	if !strings.Contains(decorated.Error(), "test error with stack") {
		t.Errorf("Error message should contain original error text")
	}

	// Error message should contain stack trace
	errorMsg := decorated.Error()
	if !strings.Contains(errorMsg, "\n\tat ") {
		t.Error("Error message should contain formatted stack trace")
	}
}

func TestDecorateWithStackTrace_WithNilError(t *testing.T) {
	t.Setenv("BAL_BACKTRACE", "true")

	decorated := DecorateWithStackTrace(nil)

	if decorated != nil {
		t.Error("Expected nil when decorating nil error")
	}
}

func TestErrorWithStackTrace_StackTrace(t *testing.T) {
	t.Setenv("BAL_BACKTRACE", "true")

	err := errors.New("test error")
	decorated := DecorateWithStackTrace(err)

	// Test through the Error() method which includes the stack trace
	errorMsg := decorated.Error()

	// Should contain original error message
	if !strings.Contains(errorMsg, "test error") {
		t.Error("Expected error message to contain original error text")
	}

	// Should contain function name
	if !strings.Contains(errorMsg, "\n\tat ") {
		t.Error("Stack trace should contain function frames")
	}

	// Should contain file path
	if !strings.Contains(errorMsg, ".go:") {
		t.Error("Stack trace should contain file paths and line numbers")
	}
}
