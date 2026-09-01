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
	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

const diagnosticSource = "ballerina"

// UTF-16 boundary: The helpers below convert core CompilerDiagnostic values
// (byte-offset-derived positions) to protocol.Diagnostic with UTF-16 character
// positions. Core must not import ls/protocol; the conversion stays here.

func convertDiagnostics(diags []compile.CompilerDiagnostic, text string) []protocol.Diagnostic {
	if len(diags) == 0 {
		return nil
	}
	lineStarts := computeLineStarts(text)
	converted := make([]protocol.Diagnostic, 0, len(diags))
	for _, diag := range diags {
		start := lineCharToUTF16Position(text, lineStarts, diag.StartLine, diag.StartChar)
		end := lineCharToUTF16Position(text, lineStarts, diag.EndLine, diag.EndChar)
		converted = append(converted, protocol.Diagnostic{
			Range:    protocol.Range{Start: start, End: end},
			Severity: protocol.NewOptional(toLSPSeverity(diag.Severity)),
			Code:     protocol.NewOptional(protocol.NewOrDiagnosticCodeString(diag.Code)),
			Source:   protocol.NewOptional(diagnosticSource),
			Message:  protocol.NewOrDiagnosticMessageString(diag.Message),
		})
	}
	return converted
}

func toLSPSeverity(severity compile.Severity) protocol.DiagnosticSeverity {
	switch severity {
	case compile.SeverityError:
		return protocol.DiagnosticSeverityError
	case compile.SeverityWarning:
		return protocol.DiagnosticSeverityWarning
	case compile.SeverityInformation:
		return protocol.DiagnosticSeverityInformation
	case compile.SeverityHint:
		return protocol.DiagnosticSeverityHint
	default:
		return protocol.DiagnosticSeverityError
	}
}

func lineCharToUTF16Position(text string, lineStarts []int, line uint32, byteChar uint32) protocol.Position {
	if int(line) >= len(lineStarts) {
		return protocol.Position{Line: line, Character: byteChar}
	}
	lineStart := lineStarts[line]
	end := lineStart + int(byteChar)
	if end > len(text) {
		end = len(text)
	}
	return protocol.Position{
		Line:      line,
		Character: utf16CodeUnits(text[lineStart:end]),
	}
}

func computeLineStarts(text string) []int {
	starts := []int{0}
	for i := 0; i < len(text); {
		switch text[i] {
		case '\r':
			if i+1 < len(text) && text[i+1] == '\n' {
				starts = append(starts, i+2)
				i += 2
			} else {
				starts = append(starts, i+1)
				i++
			}
		case '\n':
			starts = append(starts, i+1)
			i++
		default:
			i++
		}
	}
	return starts
}

func utf16CodeUnits(s string) uint32 {
	var count uint32
	for _, r := range s {
		count++
		if r >= 0x10000 {
			count++
		}
	}
	return count
}
