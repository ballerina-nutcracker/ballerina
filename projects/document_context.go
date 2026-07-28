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

package projects

import (
	"strings"
	"sync"
	"time"

	common "github.com/ballerina-nutcracker/ballerina/common"
	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/st"
	"github.com/ballerina-nutcracker/ballerina/tools/text"
)

// documentContext holds internal state for a Document.
// It manages lazy loading of syntax tree and text document.
type documentContext struct {
	documentConfig    DocumentConfig
	name              string
	diagKeyPrefix     string // composes with name to produce a unique DiagnosticEnv key
	disableSyntaxTree bool

	// Lazy-loaded with sync.Once
	syntaxTree     *st.SyntaxTree
	textDocument   text.TextDocument
	syntaxTreeOnce sync.Once
	textDocOnce    sync.Once
}

// newDocumentContext creates a documentContext from DocumentConfig.
// diagKeyPrefix is prepended to the bare name to form a globally-unique key used
// by DiagnosticEnv and SyntaxTree.FilePath. Pass "" for current-package files.
func newDocumentContext(documentConfig DocumentConfig, disableSyntaxTree bool, diagKeyPrefix string) *documentContext {
	return &documentContext{
		documentConfig:    documentConfig,
		name:              documentConfig.Name(),
		diagKeyPrefix:     diagKeyPrefix,
		disableSyntaxTree: disableSyntaxTree,
	}
}

// documentID returns the document identifier.
func (d *documentContext) documentID() DocumentID {
	return d.documentConfig.DocumentID()
}

// getName returns the bare document filename (Document.Name()).
func (d *documentContext) getName() string {
	return d.name
}

// registrationKey returns the unique key used by DiagnosticEnv and stored in
// SyntaxTree.FilePath. For current-package files this is just the bare name;
// for external dependencies it is "<org>/<moduleName>/<version>::<name>";
// for workspace members it is "<member>/<name>".
func (d *documentContext) registrationKey() string {
	if d.diagKeyPrefix == "" {
		return d.name
	}
	return d.diagKeyPrefix + d.name
}

// parseContent parses the content string and returns a SyntaxTree.
func (d *documentContext) parseContent(cx *compilercontext.CompilerContext, content string) *st.SyntaxTree {
	var syntaxTree *st.SyntaxTree
	parse := func() {
		syntaxTree, _ = parser.GetSyntaxTree(cx, d.registrationKey(), content)
	}
	if d.diagKeyPrefix != "" {
		common.WithSuppressedDebug(parse)
	} else {
		parse()
	}
	return syntaxTree
}

// parseWithStats parses the document and returns the syntax tree.
// Uses lazy loading with sync.Once for memoization when disableSyntaxTree is false.
// When disableSyntaxTree is true, parsing happens on every call (no caching).
func (d *documentContext) parseWithStats(cx *compilercontext.CompilerContext) *st.SyntaxTree {
	if d.disableSyntaxTree {
		// Parse every time without caching
		start := time.Now()
		syntaxTree := d.parseContent(cx, d.content())
		recordParseDuration(cx, time.Since(start))
		return syntaxTree
	}

	d.syntaxTreeOnce.Do(func() {
		start := time.Now()
		d.syntaxTree = d.parseContent(cx, d.content())
		recordParseDuration(cx, time.Since(start))
	})
	return d.syntaxTree
}

func recordParseDuration(cx *compilercontext.CompilerContext, duration time.Duration) {
	if cx != nil {
		cx.RecordStageDuration(compilercontext.StageParse, duration)
	}
}

// getTextDocument returns the text document (lazy loaded).
// When disableSyntaxTree is false, the text document is cached.
func (d *documentContext) getTextDocument() text.TextDocument {
	if d.disableSyntaxTree {
		return d.getTextDocumentInternal()
	}

	d.textDocOnce.Do(func() {
		d.textDocument = d.getTextDocumentInternal()
	})
	return d.textDocument
}

// getTextDocumentInternal creates a TextDocument from the document content.
func (d *documentContext) getTextDocumentInternal() text.TextDocument {
	return text.TextDocumentFromText(d.content())
}

// content returns the document content from the config.
func (d *documentContext) content() string {
	return d.documentConfig.Content()
}

// getDocumentConfig returns the underlying document configuration.
func (d *documentContext) getDocumentConfig() DocumentConfig {
	return d.documentConfig
}

// duplicate creates a copy of the context (for modification).
// The duplicated context preserves the disableSyntaxTree setting.
func (d *documentContext) duplicate() *documentContext {
	return &documentContext{
		documentConfig:    d.documentConfig,
		name:              d.name,
		diagKeyPrefix:     d.diagKeyPrefix,
		disableSyntaxTree: d.disableSyntaxTree,
	}
}

func (d *documentContext) moduleLoadRequests(cx *compilercontext.CompilerContext) []*moduleLoadRequest {
	syntaxTree := d.parseWithStats(cx)
	if syntaxTree == nil {
		return nil
	}

	rootNode := syntaxTree.RootNode
	if rootNode == nil {
		return nil
	}

	modulePart, ok := rootNode.(*st.ModulePart)
	if !ok {
		return nil
	}

	var requests []*moduleLoadRequest
	imports := modulePart.Imports()
	for i := 0; i < imports.Size(); i++ {
		importDecl := imports.Get(i)
		request := extractModuleLoadRequest(importDecl)
		if request != nil {
			requests = append(requests, request)
		}
	}
	return requests
}

func extractModuleLoadRequest(importDecl *st.ImportDeclarationNode) *moduleLoadRequest {
	// Get organization name (optional)
	var orgName *PackageOrg
	if importDecl.OrgName() != nil {
		orgNameNode := importDecl.OrgName()
		if orgNameNode.OrgName() != nil {
			// Handle quoted identifiers - strip the leading ' character
			text := orgNameNode.OrgName().Text()
			if len(text) > 0 && text[0] == '\'' {
				text = text[1:]
			}
			org := NewPackageOrg(text)
			orgName = &org
		}
	}

	// Build module name by joining identifiers with "."
	// Use Iterator which filters out separator tokens (dots)
	moduleNameList := importDecl.ModuleName()
	var moduleNameParts []string
	for ident := range moduleNameList.Iterator() {
		// Handle quoted identifiers - strip the leading ' character
		text := ident.Text()
		if len(text) > 0 && text[0] == '\'' {
			text = text[1:]
		}
		moduleNameParts = append(moduleNameParts, text)
	}
	moduleName := strings.Join(moduleNameParts, ".")

	return newModuleLoadRequest(orgName, moduleName)
}
