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
	"ballerina/ast"
	"ballerina/context"
	"ballerina/model"
	"fmt"
	"html"
	"sort"
	"strings"
)

// CFGDotExporter exports a PackageCFG in DOT format (Graphviz)
type CFGDotExporter struct {
	ctx    *context.CompilerContext
	buffer strings.Builder
}

// NewCFGDotExporter creates a new CFG DOT exporter
func NewCFGDotExporter(ctx *context.CompilerContext) *CFGDotExporter {
	return &CFGDotExporter{
		ctx: ctx,
	}
}

// Export generates a DOT format representation of the CFG
func (e *CFGDotExporter) Export(cfg *PackageCFG) string {
	e.buffer.Reset()

	// Start digraph
	e.buffer.WriteString("digraph CFG {\n")
	e.buffer.WriteString("    rankdir=TB;\n")
	e.buffer.WriteString("    node [shape=box, style=filled, fillcolor=lightgray, fontname=\"Courier\"];\n\n")

	// Sort functions by name for deterministic output
	type fnEntry struct {
		ref  model.SymbolRef
		name string
		cfg  functionCFG
	}

	var entries []fnEntry
	for ref, fcfg := range cfg.allFunctionCfgs {
		name := e.ctx.SymbolName(ref)
		entries = append(entries, fnEntry{ref: ref, name: name, cfg: *fcfg})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	// Export each function's CFG
	for _, entry := range entries {
		e.exportFunctionCFG(entry.name, entry.cfg)
	}

	e.buffer.WriteString("}\n")
	return e.buffer.String()
}

// exportFunctionCFG exports the CFG for a single function as a subgraph
func (e *CFGDotExporter) exportFunctionCFG(funcName string, cfg functionCFG) {
	// Create subgraph for this function
	clusterName := fmt.Sprintf("cluster_%s", funcName)
	fmt.Fprintf(&e.buffer, "    subgraph %s {\n", clusterName)
	fmt.Fprintf(&e.buffer, "        label=\"function: %s\";\n", html.EscapeString(funcName))
	e.buffer.WriteString("        style=rounded;\n")
	e.buffer.WriteString("        bgcolor=white;\n\n")

	// Export nodes (basic blocks)
	for _, bb := range cfg.bbs {
		e.exportBasicBlock(funcName, &bb)
	}

	e.buffer.WriteString("\n")

	// Export edges
	for _, bb := range cfg.bbs {
		e.exportEdges(funcName, &bb)
	}

	e.buffer.WriteString("    }\n\n")
}

// exportBasicBlock exports a single basic block as a node
func (e *CFGDotExporter) exportBasicBlock(funcName string, bb *basicBlock) {
	nodeID := fmt.Sprintf("%s_bb%d", funcName, bb.id)

	// Determine fill color based on block type
	fillColor := "lightgray"
	if bb.id == 0 {
		// Entry block
		fillColor = "lightgreen"
	} else if len(bb.children) == 0 {
		// Terminal block
		fillColor = "lightcoral"
	}

	// Create HTML-like label with table
	fmt.Fprintf(&e.buffer, "        %s [label=<\n", nodeID)
	e.buffer.WriteString("            <TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\" CELLPADDING=\"4\">\n")

	// Header row with block ID
	fmt.Fprintf(&e.buffer, "                <TR><TD BGCOLOR=\"%s\" COLSPAN=\"1\"><B>bb%d</B></TD></TR>\n", fillColor, bb.id)

	// AST nodes in the block
	if len(bb.nodes) > 0 {
		for _, node := range bb.nodes {
			if blangNode, ok := node.(ast.BLangNode); ok {
				nodeStr := e.formatNode(blangNode)
				fmt.Fprintf(&e.buffer, "                <TR><TD ALIGN=\"LEFT\">%s</TD></TR>\n", nodeStr)
			}
		}
	} else {
		// Empty block
		e.buffer.WriteString("                <TR><TD ALIGN=\"LEFT\"><I>(empty)</I></TD></TR>\n")
	}

	e.buffer.WriteString("            </TABLE>\n")
	fmt.Fprintf(&e.buffer, "        >, fillcolor=%s];\n", fillColor)
}

// formatNode formats an AST node for display in the DOT label
func (e *CFGDotExporter) formatNode(node ast.BLangNode) string {
	printer := &ast.PrettyPrinter{}
	nodeStr := printer.Print(node)

	// Escape HTML special characters
	nodeStr = html.EscapeString(nodeStr)

	// Replace newlines with <BR/> for multi-line nodes
	nodeStr = strings.ReplaceAll(nodeStr, "\n", "<BR ALIGN=\"LEFT\"/>")

	// Limit line length for readability
	lines := strings.Split(nodeStr, "<BR ALIGN=\"LEFT\"/>")
	var formatted []string
	for _, line := range lines {
		if len(line) > 80 {
			// Truncate long lines
			formatted = append(formatted, line[:77]+"...")
		} else {
			formatted = append(formatted, line)
		}
	}

	return strings.Join(formatted, "<BR ALIGN=\"LEFT\"/>")
}

// exportEdges exports edges from a basic block to its children
func (e *CFGDotExporter) exportEdges(funcName string, bb *basicBlock) {
	sourceID := fmt.Sprintf("%s_bb%d", funcName, bb.id)

	for i, childID := range bb.children {
		targetID := fmt.Sprintf("%s_bb%d", funcName, childID)

		// Add edge labels for conditional branches
		label := ""
		if len(bb.children) == 2 {
			// Binary branch (if/else)
			if i == 0 {
				label = " [label=\"true\"]"
			} else {
				label = " [label=\"false\"]"
			}
		} else if len(bb.children) > 2 {
			// Multiple branches (could be match statement in future)
			label = fmt.Sprintf(" [label=\"case %d\"]", i)
		}

		fmt.Fprintf(&e.buffer, "        %s -> %s%s;\n", sourceID, targetID, label)
	}
}
