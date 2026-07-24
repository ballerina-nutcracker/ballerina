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
package common

import "ballerina/tools/diagnostics"

type DiagnosticWarningCode struct {
	diagnosticId string
	messageKey   string
}

var (
	// The member represents a generic syntax warning
	// We should use this only when we can't figure out the exact warning
	WARNING_SYNTAX_WARNING = DiagnosticWarningCode{diagnosticId: "BCE10000", messageKey: "warning.syntax.warning"}

	// Missing tokens in documentation
	WARNING_MISSING_HASH_TOKEN            = DiagnosticWarningCode{diagnosticId: "BCE10001", messageKey: "warning.missing.hash.token"}
	WARNING_MISSING_SINGLE_BACKTICK_TOKEN = DiagnosticWarningCode{diagnosticId: "BCE10002", messageKey: "warning.missing.single.backtick.token"}
	WARNING_MISSING_DOUBLE_BACKTICK_TOKEN = DiagnosticWarningCode{diagnosticId: "BCE10003", messageKey: "warning.missing.double.backtick.token"}
	WARNING_MISSING_TRIPLE_BACKTICK_TOKEN = DiagnosticWarningCode{diagnosticId: "BCE10004", messageKey: "warning.missing.triple.backtick.token"}
	WARNING_MISSING_IDENTIFIER_TOKEN      = DiagnosticWarningCode{diagnosticId: "BCE10005", messageKey: "warning.missing.identifier.token"}
	WARNING_MISSING_OPEN_PAREN_TOKEN      = DiagnosticWarningCode{diagnosticId: "BCE10006", messageKey: "warning.missing.open.paren.token"}
	WARNING_MISSING_CLOSE_PAREN_TOKEN     = DiagnosticWarningCode{diagnosticId: "BCE10007", messageKey: "warning.missing.close.paren.token"}
	WARNING_MISSING_HYPHEN_TOKEN          = DiagnosticWarningCode{diagnosticId: "BCE10008", messageKey: "warning.missing.hyphen.token"}
	WARNING_MISSING_PARAMETER_NAME        = DiagnosticWarningCode{diagnosticId: "BCE10009", messageKey: "warning.missing.parameter.name"}
	WARNING_MISSING_CODE_REFERENCE        = DiagnosticWarningCode{diagnosticId: "BCE10010", messageKey: "warning.missing.code.reference"}

	// Invalid nodes in documentation
	WARNING_INVALID_BALLERINA_NAME_REFERENCE                             = DiagnosticWarningCode{diagnosticId: "BCE10100", messageKey: "warning.invalid.ballerina.name.reference"}
	WARNING_CANNOT_HAVE_DOCUMENTATION_INLINE_WITH_A_CODE_REFERENCE_BLOCK = DiagnosticWarningCode{diagnosticId: "BCE10101", messageKey: "warning.cannot.have.documentation.inline.with.a.code.reference.block"}
	WARNING_INVALID_ESCAPE_SEQUENCE                                      = DiagnosticWarningCode{diagnosticId: "BCE10102", messageKey: "warning.invalid.escape.sequence"}
)

func (d *DiagnosticWarningCode) DiagnosticId() string {
	return d.diagnosticId
}

func (d *DiagnosticWarningCode) MessageKey() string {
	return d.messageKey
}

func (d *DiagnosticWarningCode) Severity() diagnostics.DiagnosticSeverity {
	return diagnostics.Warning
}
