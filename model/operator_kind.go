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

package model

// PR-TODO: fix names
type OperatorKind string

const (
	OperatorKind_ADD                          OperatorKind = "+"
	OperatorKind_SUB                          OperatorKind = "-"
	OperatorKind_MUL                          OperatorKind = "*"
	OperatorKind_DIV                          OperatorKind = "/"
	OperatorKind_MOD                          OperatorKind = "%"
	OperatorKind_AND                          OperatorKind = "&&"
	OperatorKind_OR                           OperatorKind = "||"
	OperatorKind_EQUAL                        OperatorKind = "=="
	OperatorKind_EQUALS                       OperatorKind = "equals"
	OperatorKind_NOT_EQUAL                    OperatorKind = "!="
	OperatorKind_GREATER_THAN                 OperatorKind = ">"
	OperatorKind_GREATER_EQUAL                OperatorKind = ">="
	OperatorKind_LESS_THAN                    OperatorKind = "<"
	OperatorKind_LESS_EQUAL                   OperatorKind = "<="
	OperatorKind_IS_ASSIGNABLE                OperatorKind = "isassignable"
	OperatorKind_NOT                          OperatorKind = "!"
	OperatorKind_LENGTHOF                     OperatorKind = "lengthof"
	OperatorKind_TYPEOF                       OperatorKind = "typeof"
	OperatorKind_UNTAINT                      OperatorKind = "untaint"
	OperatorKind_INCREMENT                    OperatorKind = "++"
	OperatorKind_DECREMENT                    OperatorKind = "--"
	OperatorKind_CHECK                        OperatorKind = "check"
	OperatorKind_CHECK_PANIC                  OperatorKind = "checkpanic"
	OperatorKind_ELVIS                        OperatorKind = "?:"
	OperatorKind_BITWISE_AND                  OperatorKind = "&"
	OperatorKind_BITWISE_OR                   OperatorKind = "|"
	OperatorKind_BITWISE_XOR                  OperatorKind = "^"
	OperatorKind_BITWISE_COMPLEMENT           OperatorKind = "~"
	OperatorKind_BITWISE_LEFT_SHIFT           OperatorKind = "<<"
	OperatorKind_BITWISE_RIGHT_SHIFT          OperatorKind = ">>"
	OperatorKind_BITWISE_UNSIGNED_RIGHT_SHIFT OperatorKind = ">>>"
	OperatorKind_CLOSED_RANGE                 OperatorKind = "..."
	OperatorKind_HALF_OPEN_RANGE              OperatorKind = "..<"
	OperatorKind_REF_EQUAL                    OperatorKind = "==="
	OperatorKind_REF_NOT_EQUAL                OperatorKind = "!=="
	OperatorKind_ANNOT_ACCESS                 OperatorKind = ".@"
)

func OperatorKindValueFrom(opValue string) OperatorKind {
	switch opValue {
	case "+":
		return OperatorKind_ADD
	case "-":
		return OperatorKind_SUB
	case "*":
		return OperatorKind_MUL
	case "/":
		return OperatorKind_DIV
	case "%":
		return OperatorKind_MOD
	case "&&":
		return OperatorKind_AND
	case "||":
		return OperatorKind_OR
	case "==":
		return OperatorKind_EQUAL
	case "equals":
		return OperatorKind_EQUALS
	case "!=":
		return OperatorKind_NOT_EQUAL
	case ">":
		return OperatorKind_GREATER_THAN
	case ">=":
		return OperatorKind_GREATER_EQUAL
	case "<":
		return OperatorKind_LESS_THAN
	case "<=":
		return OperatorKind_LESS_EQUAL
	case "isassignable":
		return OperatorKind_IS_ASSIGNABLE
	case "!":
		return OperatorKind_NOT
	case "lengthof":
		return OperatorKind_LENGTHOF
	case "typeof":
		return OperatorKind_TYPEOF
	case "untaint":
		return OperatorKind_UNTAINT
	case "++":
		return OperatorKind_INCREMENT
	case "--":
		return OperatorKind_DECREMENT
	case "check":
		return OperatorKind_CHECK
	case "checkpanic":
		return OperatorKind_CHECK_PANIC
	case "?:":
		return OperatorKind_ELVIS
	case "&":
		return OperatorKind_BITWISE_AND
	case "|":
		return OperatorKind_BITWISE_OR
	case "^":
		return OperatorKind_BITWISE_XOR
	case "~":
		return OperatorKind_BITWISE_COMPLEMENT
	case "<<":
		return OperatorKind_BITWISE_LEFT_SHIFT
	case ">>":
		return OperatorKind_BITWISE_RIGHT_SHIFT
	case ">>>":
		return OperatorKind_BITWISE_UNSIGNED_RIGHT_SHIFT
	case "...":
		return OperatorKind_CLOSED_RANGE
	case "..<":
		return OperatorKind_HALF_OPEN_RANGE
	case "===":
		return OperatorKind_REF_EQUAL
	case "!==":
		return OperatorKind_REF_NOT_EQUAL
	case ".@":
		return OperatorKind_ANNOT_ACCESS
	default:
		panic("Unsupported operator: " + opValue)
	}
}
