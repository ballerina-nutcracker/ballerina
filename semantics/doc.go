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

// Package semantics provides API for performing semantic anlysis
// It provides fallowing primitive operation
//  1. ResolveSymbols
//  2. ResolvePublicNodeTypes
//  3. ResolvePrivateNodesTypes
//  4. AnalyzeSemantics
//  5. CreateControlFlowGraph
//  6. AnalyzeCFG
//
// All operations execpt ResolveSymbols are performed on package nodes
// (as far frontend is concerned as both ballerina packages and modules are the same)
// After stages 1 and 2 you can consider the public inteface represented by ExportedSymbolSpace to
// be stable. That is once you have performed these steps on the dependencies of a given package you
// can perform other analysis operations concurrentlly with any other package.
//
// Each operation indicate errors using complier context. If one stage reports an error you shouldn't
// proceed to the next stage.
//
// Stage 1 - 3 works with both normal and error recovering ASTs.
package semantics
