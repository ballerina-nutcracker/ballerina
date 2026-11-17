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

// Names and prefixes for compiler-generated functions. A source name can never
// start with '$', so every generated name built from these must start with '$'
// to never clash with a source name. Registering a name here does not decide
// whether the runtime hides its frame from stack traces.
const (
	// DefaultParamFunctionPrefix marks the function that evaluates a
	// defaultable parameter (`semantics/internal/symbols`).
	DefaultParamFunctionPrefix = "$default$"
	// AnonFunctionPrefix marks an anonymous function (`context`).
	AnonFunctionPrefix = "$anonFunc$"
	// WorkerClosurePrefix marks a named worker's body closure (`desugar`).
	WorkerClosurePrefix = "$worker:"

	// ModuleStartFunctionName, ModuleGracefulStopFunctionName and
	// ModuleImmediateStopFunctionName name the per-module lifecycle dispatch
	// functions (`desugar`). Each one iterates the module's listener array and
	// invokes the matching listener method (see desugar.ListenerMethodFor).
	ModuleStartFunctionName         = "$start"
	ModuleGracefulStopFunctionName  = "$gracefulStop"
	ModuleImmediateStopFunctionName = "$immediateStop"

	// XMLStepFunctionPrefix starts the name of every function an XML step
	// lowers to (`semantics/internal/types`).
	XMLStepFunctionPrefix = "$xmlStep$"

	// WorkerBodyFunctionSuffix names the hidden function a worker's body runs
	// in when the worker waits for the delivery of its async messages before
	// terminating (`desugar`).
	WorkerBodyFunctionSuffix = "$body"

	RemoteMethodFunctionPrefix   = "$remote$"
	ResourceMethodFunctionPrefix = "$resource$"
	MonomorphicFunctionPrefix    = "$mono$"
	XMLStepIndexFunctionName     = "$stepIndex"
	ListenerMethodFunctionPrefix = "$listener$"
)

// DefaultWorkerName is the name of the default worker of a block function body
// with named workers. Source code can't produce it.
const DefaultWorkerName = "$function"

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
