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

package semantics

import (
	"ballerina-lang-go/ast"
	"ballerina-lang-go/context"
	"fmt"
	"sort"
	"strings"
)

// CFGPrettyPrinter prints a PackageCFG in a human-readable format
type CFGPrettyPrinter struct {
	ctx    *context.CompilerContext
	buffer strings.Builder
}

// NewCFGPrettyPrinter creates a new CFG pretty printer
func NewCFGPrettyPrinter(ctx *context.CompilerContext) *CFGPrettyPrinter {
	return &CFGPrettyPrinter{
		ctx: ctx,
	}
}

// Print generates a string representation of the CFG
func (p *CFGPrettyPrinter) Print(cfg *PackageCFG) string {
	p.buffer.Reset()

	type fnEntry struct {
		name string
		cfg  functionCFG
	}

	type classEntry struct {
		name    string
		initCfg *functionCFG
		methods []fnEntry
	}

	var classes []classEntry
	for classRef, classMethods := range cfg.methodCfgs {
		ce := classEntry{name: p.ctx.SymbolName(classRef)}
		for methodRef, fcfg := range classMethods {
			name := p.ctx.SymbolName(methodRef)
			if name == "init" {
				ce.initCfg = &fcfg
				continue
			}
			ce.methods = append(ce.methods, fnEntry{name: name, cfg: fcfg})
		}
		sort.Slice(ce.methods, func(i, j int) bool {
			return ce.methods[i].name < ce.methods[j].name
		})
		classes = append(classes, ce)
	}
	sort.Slice(classes, func(i, j int) bool {
		return classes[i].name < classes[j].name
	})

	// Collect top-level functions
	var topLevel []fnEntry
	for ref, fnCfg := range cfg.funcCfgs {
		topLevel = append(topLevel, fnEntry{name: p.ctx.SymbolName(ref), cfg: fnCfg})
	}
	sort.Slice(topLevel, func(i, j int) bool {
		return topLevel[i].name < topLevel[j].name
	})

	printed := 0
	for _, ce := range classes {
		if ce.initCfg == nil && len(ce.methods) == 0 {
			continue
		}
		if printed > 0 {
			p.buffer.WriteString("\n")
		}
		p.buffer.WriteString("(")
		p.buffer.WriteString(ce.name)
		p.buffer.WriteString("\n")
		if ce.initCfg != nil {
			p.printFunctionCFG("init", *ce.initCfg, 2)
			p.buffer.WriteString("\n")
		}
		for _, me := range ce.methods {
			p.printFunctionCFG(me.name, me.cfg, 2)
			p.buffer.WriteString("\n")
		}
		p.buffer.WriteString(")")
		printed++
	}

	for _, entry := range topLevel {
		if printed > 0 {
			p.buffer.WriteString("\n")
		}
		p.printFunctionCFG(entry.name, entry.cfg, 0)
		printed++
	}

	return p.buffer.String()
}

func (p *CFGPrettyPrinter) printFunctionCFG(funcName string, cfg functionCFG, indent int) {
	prefix := strings.Repeat(" ", indent)
	p.buffer.WriteString(prefix)
	p.buffer.WriteString("(")
	p.buffer.WriteString(funcName)
	p.buffer.WriteString("\n")

	for _, bb := range cfg.bbs {
		p.printBasicBlock(&bb, indent)
	}

	p.buffer.WriteString(prefix)
	p.buffer.WriteString(")")
}

func (p *CFGPrettyPrinter) printBasicBlock(bb *basicBlock, indent int) {
	prefix := strings.Repeat(" ", indent+2)
	nodePrefix := strings.Repeat(" ", indent+4)

	p.buffer.WriteString(prefix)
	p.buffer.WriteString("(bb")
	fmt.Fprintf(&p.buffer, "%d", bb.id)
	p.buffer.WriteString(" ")

	// Print parents
	p.buffer.WriteString("(")
	for i, parent := range bb.parents {
		if i > 0 {
			p.buffer.WriteString(" ")
		}
		fmt.Fprintf(&p.buffer, "bb%d", parent)
	}
	p.buffer.WriteString(")")
	p.buffer.WriteString(" ")

	// Print children
	p.buffer.WriteString("(")
	for i, child := range bb.children {
		if i > 0 {
			p.buffer.WriteString(" ")
		}
		fmt.Fprintf(&p.buffer, "bb%d", child)
	}
	p.buffer.WriteString(")")

	// Print nodes if any
	if len(bb.nodes) > 0 {
		p.buffer.WriteString("\n")
		p.printNodes(bb.nodes, nodePrefix)
		p.buffer.WriteString(prefix)
	}

	p.buffer.WriteString(")\n")
}

func (p *CFGPrettyPrinter) printNodes(nodes []ast.Node, prefix string) {
	for _, node := range nodes {
		if blangNode, ok := node.(ast.BLangNode); ok {
			printer := &ast.PrettyPrinter{}
			nodeStr := printer.Print(blangNode)

			lines := strings.SplitSeq(nodeStr, "\n")
			for line := range lines {
				if line != "" {
					p.buffer.WriteString(prefix)
					p.buffer.WriteString(line)
					p.buffer.WriteString("\n")
				}
			}
		}
	}
}
