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

package constants

const (
	Underscore = "_"
	UserHome   = "user.home"
)

// Prefixes of the lookup keys the compiler hands to the functions it
// generates. A source name can never start with '$', so a key carrying one of
// these prefixes is always a generated function. The runtime matches on them
// to keep generated frames out of stack traces, so a new kind of generated
// function belongs here as well as at its generation site.
const (
	// DefaultParamFunctionPrefix marks the function that evaluates a
	// defaultable parameter (`semantics/internal/symbols`).
	DefaultParamFunctionPrefix = "$default$"
	// AnonFunctionPrefix marks an anonymous function (`context`).
	AnonFunctionPrefix = "$anonFunc$"
	// WorkerClosurePrefix marks a named worker's body closure (`desugar`).
	WorkerClosurePrefix = "$worker:"
)

type SymbolFlag int64

const (
	PUBLIC SymbolFlag = 1 << iota
	NATIVE
	FINAL
	ATTACHED
	READONLY
	REQUIRED
	PRIVATE
	OPTIONAL
	REMOTE
	CLIENT
	RESOURCE
	SERVICE
	TRANSACTIONAL
	CLASS
	ISOLATED
	ENUM
	ANY_FUNCTION
)

func (sf SymbolFlag) IsOn(flag SymbolFlag) bool {
	return (sf & flag) == flag
}
