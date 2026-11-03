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

package ast

import "github.com/ballerina-nutcracker/ballerina/model"

type BLangNode any

type TypeData struct {
	Ty int
}

type Visitor interface {
	Visit(node BLangNode) (w Visitor)
	VisitTypeData(typeData *TypeData) (w Visitor)
}

func Walk(v Visitor, node BLangNode) {}

func WalkTypeData(v Visitor, typeData *TypeData) {}

func SetType(node BLangNode, ty int) {}

type Block struct {
	Stmts []int
}

type Node struct {
	Attrs map[string]int
	Data  TypeData
	Ty    int
	Args  []int
	Count int
	Body  Block
}

func (n *Node) SetSymbol(symbol int) {}

func (n *Node) Symbol() model.SymbolRef { return model.SymbolRef{} }

func (n *Node) AddArg(arg int) {}
