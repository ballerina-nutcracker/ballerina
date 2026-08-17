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

// Package compilerplugin defines the public compiler plugin contract.
package compilerplugin

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
)

// Stage identifies a compiler stage after which a plugin runs.
type Stage uint8

const (
	// AfterSemantics runs after CFG analysis and immediately before desugaring.
	AfterSemantics Stage = iota
)

// PackageTransformer transforms or validates a semantically analyzed package AST.
type PackageTransformer func(
	*context.CompilerContext,
	model.ExportedSymbolSpace,
	*ast.BLangPackage,
) (*ast.BLangPackage, error)

// CompilerPlugin declares a package transformer and its execution stage.
type CompilerPlugin struct {
	After              Stage
	PackageTransformer PackageTransformer
}
