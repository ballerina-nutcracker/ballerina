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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package server

import (
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

func TestConvertDiagnosticsUTF16Boundary(t *testing.T) {
	text := "public function main() {\n    int x = 1;\n}\n"
	diags := []compile.CompilerDiagnostic{
		{
			StartLine: 1,
			StartChar: 4,
			EndLine:   1,
			EndChar:   14,
			Severity:  compile.SeverityError,
			Code:      "SEMANTIC_ERROR",
			Message:   "unused variable 'x'",
		},
	}
	converted := convertDiagnostics(diags, text)
	if len(converted) != 1 {
		t.Fatalf("converted = %d, want 1", len(converted))
	}
	diag := converted[0]
	severity, _ := diag.Severity.Value()
	if severity != protocol.DiagnosticSeverityError {
		t.Errorf("severity = %v, want SeverityError", severity)
	}
	source, _ := diag.Source.Value()
	if source != "ballerina" {
		t.Errorf("source = %q, want ballerina", source)
	}
	code, _ := diag.Code.Value()
	codeStr, _ := code.String()
	if codeStr != "SEMANTIC_ERROR" {
		t.Errorf("code = %q, want SEMANTIC_ERROR", codeStr)
	}
	wantStart := protocol.Position{Line: 1, Character: 4}
	wantEnd := protocol.Position{Line: 1, Character: 14}
	if diag.Range.Start != wantStart {
		t.Errorf("range start = %+v, want %+v", diag.Range.Start, wantStart)
	}
	if diag.Range.End != wantEnd {
		t.Errorf("range end = %+v, want %+v", diag.Range.End, wantEnd)
	}
}

func TestConvertDiagnosticsEmptyReturnsNil(t *testing.T) {
	if converted := convertDiagnostics(nil, "abc"); converted != nil {
		t.Fatalf("convertDiagnostics(nil) = %v, want nil", converted)
	}
}

func TestConvertDiagnosticsSurrogatePairUTF16(t *testing.T) {
	text := "😀x\n"
	// 😀 is 4 bytes (1 UTF-16 code unit pair = 2), x is 1 byte (1 UTF-16 unit)
	// Byte offset 4 = after 😀, char 2 in UTF-16
	diags := []compile.CompilerDiagnostic{
		{
			StartLine: 0,
			StartChar: 4, // byte offset 4 = after 😀
			EndLine:   0,
			EndChar:   5, // byte offset 5 = after x
			Severity:  compile.SeverityError,
			Code:      "TEST",
			Message:   "test",
		},
	}
	converted := convertDiagnostics(diags, text)
	if len(converted) != 1 {
		t.Fatalf("converted = %d, want 1", len(converted))
	}
	wantStart := protocol.Position{Line: 0, Character: 2}
	wantEnd := protocol.Position{Line: 0, Character: 3}
	if converted[0].Range.Start != wantStart {
		t.Errorf("start = %+v, want %+v", converted[0].Range.Start, wantStart)
	}
	if converted[0].Range.End != wantEnd {
		t.Errorf("end = %+v, want %+v", converted[0].Range.End, wantEnd)
	}
}
