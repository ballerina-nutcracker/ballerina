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

package symbols

import (
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

func xmlnsPrefixName(prefix string) string {
	if prefix == "" {
		return model.DefaultXMLNSSymbolName
	}
	return prefix
}

func defineXMLNS(resolver symbolResolver, scope model.Scope, prefix, uri string, pos diagnostics.Location) (model.SymbolRef, bool) {
	if uri == "" {
		resolver.GetCtx().SemanticError("XML namespace URI cannot be empty", pos)
		return model.SymbolRef{}, false
	}
	return declareXMLNS(resolver, scope, prefix, uri, pos)
}

func declareXMLNS(resolver symbolResolver, scope model.Scope, prefix, uri string, pos diagnostics.Location) (model.SymbolRef, bool) {
	ensurePrefixMap(resolver, scope)
	name := xmlnsPrefixName(prefix)
	if localXMLNSPrefixExists(scope, name) {
		switch prefix {
		case model.XMLNSReservedPrefix:
			resolver.GetCtx().SemanticError("cannot redeclare reserved XML namespace prefix 'xmlns'", pos)
		case "":
			resolver.GetCtx().SemanticError("default XML namespace already declared in this scope", pos)
		default:
			resolver.GetCtx().SemanticError("XML namespace prefix '"+prefix+"' already declared in this scope", pos)
		}
		return model.SymbolRef{}, false
	}
	if localPrefixExists(scope, name) {
		resolver.GetCtx().SemanticError("redeclared symbol '"+name+"'", pos)
		return model.SymbolRef{}, false
	}
	return defineXMLNSSymbol(resolver, scope, name, uri, pos), true
}

func ensurePrefixMap(resolver symbolResolver, scope model.Scope) {
	switch s := scope.(type) {
	case *model.ModuleScope:
		if s.Prefix == nil {
			s.Prefix = make(map[string]model.ExportedSymbolSpace)
		}
		if _, ok := s.Prefix[model.XMLNSReservedPrefix]; !ok {
			defineXMLNSSymbol(resolver, s, model.XMLNSReservedPrefix, model.XMLNSReservedURI, diagnostics.NewBuiltinLocation())
		}
	case *model.BlockScope:
		if s.Prefix == nil {
			s.Prefix = make(map[string]model.ExportedSymbolSpace)
		}
	case *model.FunctionScope:
		if s.Prefix == nil {
			s.Prefix = make(map[string]model.ExportedSymbolSpace)
		}
	case *xmlnsChildScope:
		if s.prefix == nil {
			s.prefix = make(map[string]model.ExportedSymbolSpace)
		}
	}
}

func defineXMLNSSymbol(resolver symbolResolver, scope model.Scope, prefix, uri string, location diagnostics.Location) model.SymbolRef {
	space := resolver.GetCtx().NewSymbolSpace(resolver.GetPkgID())
	space.AddSymbol(prefix, model.NewXMLNSSymbol(prefix, uri, location))
	exported := model.NewExportedSymbolSpaces([]*model.SymbolSpace{space}, nil)
	setLocalPrefix(scope, prefix, exported)
	ref, _ := exported.GetSymbol(prefix)
	return ref
}

func setLocalPrefix(scope model.Scope, prefix string, exported model.ExportedSymbolSpace) {
	switch s := scope.(type) {
	case *model.ModuleScope:
		s.Prefix[prefix] = exported
	case *model.BlockScope:
		s.Prefix[prefix] = exported
	case *model.FunctionScope:
		s.Prefix[prefix] = exported
	case *xmlnsChildScope:
		s.prefix[prefix] = exported
	}
}

func localXMLNSPrefixExists(scope model.Scope, prefix string) bool {
	exported, ok := localPrefixSpace(scope, prefix)
	if !ok {
		return false
	}
	_, ok = exported.GetSymbol(prefix)
	return ok
}

func localPrefixExists(scope model.Scope, prefix string) bool {
	_, ok := localPrefixSpace(scope, prefix)
	return ok
}

func localPrefixSpace(scope model.Scope, prefix string) (model.ExportedSymbolSpace, bool) {
	switch s := scope.(type) {
	case *model.ModuleScope:
		exported, ok := s.Prefix[prefix]
		return exported, ok
	case *model.BlockScope:
		exported, ok := s.Prefix[prefix]
		return exported, ok
	case *model.FunctionScope:
		exported, ok := s.Prefix[prefix]
		return exported, ok
	case *xmlnsChildScope:
		exported, ok := s.prefix[prefix]
		return exported, ok
	}
	return model.ExportedSymbolSpace{}, false
}

func lookupXMLNS(scope model.Scope, prefix string) (model.SymbolRef, model.Scope, bool) {
	if exported, ok := localPrefixSpace(scope, prefix); ok {
		ref, ok := exported.GetSymbol(prefix)
		return ref, scope, ok
	}
	switch s := scope.(type) {
	case *model.ModuleScope:
		return model.SymbolRef{}, nil, false
	case *model.BlockScope:
		return lookupXMLNS(s.Parent, prefix)
	case *model.FunctionScope:
		return lookupXMLNS(s.Parent, prefix)
	case *xmlnsChildScope:
		return lookupXMLNS(s.parent, prefix)
	}
	return model.SymbolRef{}, nil, false
}

func processCompilationUnitXMLNS(resolver *compilationUnitSymbolResolver, cu *ast.BLangCompilationUnit) {
	for _, node := range cu.TopLevelNodes {
		decl, ok := node.(*ast.BLangXMLNS)
		if !ok {
			continue
		}
		processXMLNSDecl(resolver, resolver.scope, decl)
	}
}

func processBlockXMLNS(resolver *blockSymbolResolver, decl *ast.BLangXMLNS) bool {
	return processXMLNSDecl(resolver, resolver.scope, decl)
}

func processXMLNSDecl(resolver symbolResolver, scope model.Scope, decl *ast.BLangXMLNS) bool {
	if decl.GetNamespaceURI() == nil {
		resolver.GetCtx().SemanticError("xmlns declaration missing URI", decl.GetPosition())
		return false
	}
	prefix := ""
	if p := decl.GetPrefix(); p != nil {
		prefix = p.GetValue()
	}
	ref, ok := declareXMLNS(resolver, scope, prefix, "", decl.GetPosition())
	if !ok {
		return false
	}
	decl.SetSymbol(ref)
	return true
}

func splitXMLName(name string) (prefix, local string) {
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		return name[:idx], name[idx+1:]
	}
	return "", name
}

func resolveXMLElementLiteralNamespaces(resolver symbolResolver, scope model.Scope, e *ast.BLangXMLElementLiteral, rootNeeds map[string]model.SymbolRef) bool {
	ensurePrefixMap(resolver, scope)
	childScope := newXMLNSChildScope(scope)
	attrs, ok := stripInlineXMLNSAttrs(resolver, childScope, e)
	e.Attrs = attrs

	name := e.LocalName
	if e.Prefix != "" {
		name = e.Prefix + ":" + e.LocalName
	}
	nsRef, nameOk := resolveXMLNameRef(resolver, childScope, name, e.GetPosition(), rootNeeds, true)
	e.NamespaceSymbol = nsRef
	ok = nameOk && ok
	for i := range e.Attrs {
		attr := &e.Attrs[i]
		attrRef, attrOk := resolveXMLNameRef(resolver, childScope, attr.Name, attr.GetPosition(), rootNeeds, false)
		attr.NamespaceSymbol = attrRef
		ok = attrOk && ok
	}

	if e.Content != nil {
		ok = resolveXMLContent(resolver, childScope, e.Content, rootNeeds) && ok
	}
	return ok
}

func resolveXMLContent(resolver symbolResolver, scope model.Scope, content ast.BLangExpression, rootNeeds map[string]model.SymbolRef) bool {
	switch c := content.(type) {
	case *ast.BLangXMLElementLiteral:
		return resolveXMLElementLiteralNamespaces(resolver, scope, c, rootNeeds)
	case *ast.BLangXMLSequenceLiteral:
		ok := true
		for _, child := range c.Children {
			ok = resolveXMLContent(resolver, scope, child, rootNeeds) && ok
		}
		return ok
	}
	return true
}

func resolveXMLNameRef(resolver symbolResolver, scope model.Scope, name string, pos diagnostics.Location, rootNeeds map[string]model.SymbolRef, isElement bool) (model.SymbolRef, bool) {
	prefix, _ := splitXMLName(name)
	if prefix == "" {
		if !isElement {
			return model.SymbolRef{}, true
		}
		ref, defScope, ok := lookupXMLNS(scope, model.DefaultXMLNSSymbolName)
		if !ok || defScope == scope {
			return ref, true
		}
		if _, fromXMLAncestor := defScope.(*xmlnsChildScope); fromXMLAncestor {
			return ref, true
		}
		rootNeeds["xmlns"] = ref
		return ref, true
	}
	ref, defScope, ok := lookupXMLNS(scope, prefix)
	if !ok {
		resolver.GetCtx().SemanticError("undefined XML namespace prefix '"+prefix+"'", pos)
		return model.SymbolRef{}, false
	}
	if defScope != scope {
		if _, fromXMLAncestor := defScope.(*xmlnsChildScope); !fromXMLAncestor {
			rootNeeds["xmlns:"+prefix] = ref
		}
	}
	return ref, true
}

func stripInlineXMLNSAttrs(resolver symbolResolver, childScope model.Scope, e *ast.BLangXMLElementLiteral) ([]ast.BLangXMLAttribute, bool) {
	kept := make([]ast.BLangXMLAttribute, 0, len(e.Attrs))
	allOk := true
	for i := range e.Attrs {
		attr := e.Attrs[i]
		prefix, local := splitXMLName(attr.Name)
		if !isXMLNSAttr(prefix, local) {
			kept = append(kept, attr)
			continue
		}
		uri, ok := xmlnsAttrURI(resolver, &attr)
		if !ok {
			allOk = false
			continue
		}
		var nsPrefix string
		if prefix == "" {
			nsPrefix = ""
		} else {
			nsPrefix = local
		}
		ref, ok := defineXMLNS(resolver, childScope, nsPrefix, uri, attr.GetPosition())
		if !ok {
			allOk = false
			continue
		}
		e.Namespaces = append(e.Namespaces, ref)
	}
	return kept, allOk
}

func isXMLNSAttr(prefix, local string) bool {
	if prefix == "" {
		return local == "xmlns"
	}
	return prefix == "xmlns"
}

func xmlnsAttrURI(resolver symbolResolver, attr *ast.BLangXMLAttribute) (string, bool) {
	if attr.Value == nil {
		resolver.GetCtx().SemanticError("xmlns attribute missing URI", attr.GetPosition())
		return "", false
	}
	lit, ok := attr.Value.(*ast.BLangLiteral)
	if !ok {
		resolver.GetCtx().SemanticError("xmlns attribute URI must be a string literal", attr.GetPosition())
		return "", false
	}
	uri, ok := lit.GetValue().(string)
	if !ok {
		resolver.GetCtx().SemanticError("xmlns attribute URI must be a string", attr.GetPosition())
		return "", false
	}
	return uri, true
}

type xmlnsChildScope struct {
	parent model.Scope
	prefix map[string]model.ExportedSymbolSpace
}

func newXMLNSChildScope(parent model.Scope) *xmlnsChildScope {
	return &xmlnsChildScope{parent: parent, prefix: make(map[string]model.ExportedSymbolSpace)}
}

func (s *xmlnsChildScope) GetSymbol(name string) (model.SymbolRef, bool) {
	return s.parent.GetSymbol(name)
}

func (s *xmlnsChildScope) GetPrefixedSymbol(prefix, name string) (model.SymbolRef, bool) {
	return s.parent.GetPrefixedSymbol(prefix, name)
}

func (s *xmlnsChildScope) AddSymbol(name string, symbol model.Symbol) {
	s.parent.AddSymbol(name, symbol)
}

var _ model.Scope = &xmlnsChildScope{}

func xmlnsDeclKey(resolver symbolResolver, symbol model.Symbol) (string, bool) {
	key, err := model.XMLNamespaceDeclKey(symbol)
	if err != nil {
		resolver.GetCtx().InternalError(err.Error(), diagnostics.Location{})
		return "", false
	}
	return key, true
}

func mergeNamespaces(resolver symbolResolver, root *ast.BLangXMLElementLiteral, extras map[string]model.SymbolRef) bool {
	ok := true
	existing := make(map[string]struct{}, len(root.Namespaces))
	for _, ref := range root.Namespaces {
		key, keyOk := xmlnsDeclKey(resolver, resolver.GetCtx().GetSymbol(ref))
		existing[key] = struct{}{}
		ok = keyOk && ok
	}
	for k, v := range extras {
		if _, exists := existing[k]; exists {
			continue
		}
		root.Namespaces = append(root.Namespaces, v)
		existing[k] = struct{}{}
	}
	return ok
}

func appendXMLNSTemplateNamespace(resolver symbolResolver, insn *ast.XMLTemplateNamespaceInsertion, seen map[string]struct{}, ref model.SymbolRef) bool {
	key, ok := xmlnsDeclKey(resolver, resolver.GetCtx().GetSymbol(ref))
	if _, exists := seen[key]; exists {
		return ok
	}
	insn.Namespaces = append(insn.Namespaces, ref)
	seen[key] = struct{}{}
	return ok
}

func resolveXMLTemplateNamespaces(resolver symbolResolver, scope model.Scope, e *ast.BLangXMLTemplateExpr) bool {
	ensurePrefixMap(resolver, scope)
	ok := true
	for stringIndex := range e.NamespaceInsertions {
		for i := range e.NamespaceInsertions[stringIndex] {
			insn := &e.NamespaceInsertions[stringIndex][i]
			seen := make(map[string]struct{}, len(insn.Namespaces))
			for _, ref := range insn.Namespaces {
				key, keyOk := xmlnsDeclKey(resolver, resolver.GetCtx().GetSymbol(ref))
				seen[key] = struct{}{}
				ok = keyOk && ok
			}
			if insn.NeedsDefaultNS {
				if ref, _, found := lookupXMLNS(scope, model.DefaultXMLNSSymbolName); found {
					ok = appendXMLNSTemplateNamespace(resolver, insn, seen, ref) && ok
				}
			}
			for prefix := range insn.UsedPrefixes {
				ref, _, found := lookupXMLNS(scope, prefix)
				if !found {
					resolver.GetCtx().SemanticError("undefined XML namespace prefix '"+prefix+"'", e.GetPosition())
					ok = false
					continue
				}
				if prefix == "" || prefix == model.XMLNSReservedPrefix {
					continue
				}
				ok = appendXMLNSTemplateNamespace(resolver, insn, seen, ref) && ok
			}
		}
	}
	return ok
}

func resolveAtomicNamePattern(resolver symbolResolver, scope model.Scope, pattern ast.BLangAtomicNamePattern) (ast.BLangAtomicNamePattern, bool) {
	switch pattern.Kind {
	case ast.NamePatternKindQualifiedIdentifier, ast.NamePatternKindPrefix:
		prefix := pattern.NamespacePrefix.GetValue()
		ref, _, ok := lookupXMLNS(scope, prefix)
		if !ok {
			resolver.GetCtx().SemanticError("undefined XML namespace prefix '"+prefix+"'", pattern.NamespacePrefix.GetPosition())
			return pattern, false
		}
		pattern.NamespaceSymbol = ref
	case ast.NamePatternKindIdentifier:
		if ref, _, ok := lookupXMLNS(scope, model.DefaultXMLNSSymbolName); ok {
			pattern.Kind = ast.NamePatternKindQualifiedIdentifier
			pattern.NamespaceSymbol = ref
		}
	case ast.NamePatternKindWildCard:
		// A wildcard matches any name, so there is no prefix to resolve.
	}
	return pattern, true
}
