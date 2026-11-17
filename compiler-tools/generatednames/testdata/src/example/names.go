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

package example

import (
	"fmt"
	names "github.com/ballerina-nutcracker/ballerina/common/constants"
	"github.com/ballerina-nutcracker/ballerina/context"
	m "github.com/ballerina-nutcracker/ballerina/model"
)

const localName = "$newKind$"       // want "generated function name must use"
func helper() string                { return "$otherKind$" } // want "generated function name must use"
func ordinary() string              { return "$ordinaryVariable$" }
func NewFunctionSymbol(name string) {}
func check() {
	m.NewFunctionSymbol("$fresh$") // want "generated function name must use"
	m.NewFunctionSymbol(localName)
	m.NewFunctionSymbol(helper())
	name := fmt.Sprintf("$formatted$%d", 1) // want "generated function name must use"
	m.NewFunctionSymbol(name)
	m.NewFunctionSymbol(fmt.Sprintf("%s%d", "$formatArg$", 1)) // want "generated function name must use"
	prefix := "$concat$"                                       // want "generated function name must use"
	m.NewFunctionSymbol(prefix + "tail")
	m.NewFunctionSymbol(names.AnonFunctionPrefix + "1")
	m.NewFunctionSymbol(names.ListenerMethodFunctionPrefix + "start")
	m.NewVariableSymbol("$variable")
	m.NewVariableSymbol(ordinary())
	NewFunctionSymbol("$notTheModelConstructor")
	_ = "$anonFunc$1"
	_ = `$worker:foo`
	_ = "$start"
	_ = "$body"
	_ = "$anonType$"
	m.NewFunctionSymbol("sourceName" + "$suffix")
	m.NewVariableSymbol("$startTemporary")
	m.NewVariableSymbol(identity("$unrelatedVariable"))
	m.NewFunctionSymbol(identity(names.AnonFunctionPrefix + "1"))
	m.NewFunctionSymbol(identity("$identityFunction"))               // want "generated function name must use"
	m.NewFunctionSymbol("" + "$anotherFunction")                     // want "generated function name must use"
	m.NewFunctionSymbol(fmt.Sprintf("%s%s", "", "$unknownFunction")) // want "generated function name must use"
	m.NewFunctionSymbol(fmt.Sprintf("%s%s", names.AnonFunctionPrefix, "$allowedSuffix"))
	const alias = names.AnonFunctionPrefix
	m.NewFunctionSymbol("" + alias + "$allowedSuffix")
	m.NewVariableSymbol(wrappedIdentity("$variableThroughNestedCalls"))
	m.NewFunctionSymbol(wrappedIdentity(names.WorkerClosurePrefix + "1"))
	functionSink(identity("$throughSinkParameter")) // want "generated function name must use"
	m.NewFunctionSymbol(reassigned(names.AnonFunctionPrefix))
	m.NewFunctionSymbol(("$parenthesised")) // want "generated function name must use"
	m.NewFunctionSymbol(helper())
	m.NewFunctionSymbol(nestedReturn()())
	m.NewFunctionSymbol(recursive("$recursive", 1))  // want "generated function name must use"
	context.CreateFunctionSymbol("$contextFunction") // want "generated function name must use"
	context.CreateVariableSymbol("$contextVariable")
}

func identity(s string) string        { return s }
func wrappedIdentity(s string) string { return identity(s) }
func functionSink(s string)           { m.NewFunctionSymbol(s) }
func reassigned(s string) string {
	s = "$assignedParameter" // want "generated function name must use"
	return s
}
func nestedReturn() func() string {
	return func() string { return "$nestedLiteral" }
}
func recursive(s string, n int) string {
	if n == 0 {
		return s
	}
	return recursive(s, n-1)
}
