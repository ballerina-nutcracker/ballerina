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
	"strings"

	"ballerina-lang-go/ast"
	"ballerina-lang-go/model"
	"ballerina-lang-go/tools/diagnostics"
)

func extractXMLNSURI[T symbolResolver](resolver T, uriExpr ast.BLangExpression, pos diagnostics.Location) (string, bool) {
	lit, ok := uriExpr.(*ast.BLangLiteral)
	if !ok {
		semanticError(resolver, "xmlns URI must be a string literal", pos)
		return "", false
	}
	val, ok := lit.GetValue().(string)
	if !ok {
		semanticError(resolver, "xmlns URI must be a string", pos)
		return "", false
	}
	return val, true
}

func xmlnsPrefixName(prefix string) string {
	if prefix == "" {
		return model.DefaultXMLNSSymbolName
	}
	return prefix
}

func defineXMLNS[T symbolResolver](resolver T, scope model.Scope, prefix, uri string, pos diagnostics.Location) (model.SymbolRef, bool) {
	ensurePrefixMap(resolver, scope)
	if uri == "" {
		semanticError(resolver, "XML namespace URI cannot be empty", pos)
		return model.SymbolRef{}, false
	}
	name := xmlnsPrefixName(prefix)
	if localXMLNSPrefixExists(scope, name) {
		switch prefix {
		case model.XMLNSReservedPrefix:
			semanticError(resolver, "cannot redeclare reserved XML namespace prefix 'xmlns'", pos)
		case "":
			semanticError(resolver, "default XML namespace already declared in this scope", pos)
		default:
			semanticError(resolver, "XML namespace prefix '"+prefix+"' already declared in this scope", pos)
		}
		return model.SymbolRef{}, false
	}
	if localPrefixExists(scope, name) {
		semanticError(resolver, "redeclared symbol '"+name+"'", pos)
		return model.SymbolRef{}, false
	}
	return defineXMLNSSymbol(resolver, scope, name, uri, pos), true
}

func ensurePrefixMap[T symbolResolver](resolver T, scope model.Scope) {
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

func defineXMLNSSymbol[T symbolResolver](resolver T, scope model.Scope, prefix, uri string, location diagnostics.Location) model.SymbolRef {
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

func processCompilationUnitXMLNS(resolver *moduleSymbolResolver, cu *ast.BLangCompilationUnit) {
	for _, node := range cu.TopLevelNodes {
		decl, ok := node.(*ast.BLangXMLNS)
		if !ok {
			continue
		}
		processXMLNSDecl(resolver, resolver.scope, decl)
	}
}

func processBlockXMLNS(resolver *blockSymbolResolver, decl *ast.BLangXMLNS) {
	processXMLNSDecl(resolver, resolver.scope, decl)
}

func processXMLNSDecl[T symbolResolver](resolver T, scope model.Scope, decl *ast.BLangXMLNS) {
	uriExpr := decl.GetNamespaceURI()
	if uriExpr == nil {
		semanticError(resolver, "xmlns declaration missing URI", decl.GetPosition())
		return
	}
	uri, ok := extractXMLNSURI(resolver, uriExpr, decl.GetPosition())
	if !ok {
		return
	}
	prefix := ""
	if p := decl.GetPrefix(); p != nil {
		prefix = p.GetValue()
	}
	defineXMLNS(resolver, scope, prefix, uri, decl.GetPosition())
}

func splitXMLName(name string) (prefix, local string) {
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		return name[:idx], name[idx+1:]
	}
	return "", name
}

func resolveXMLElementLiteralNamespaces[T symbolResolver](resolver T, scope model.Scope, e *ast.BLangXMLElementLiteral, rootNeeds map[string]model.SymbolRef) {
	ensurePrefixMap(resolver, scope)
	childScope := newXMLNSChildScope(scope)
	e.Attrs = stripInlineXMLNSAttrs(resolver, childScope, e)

	resolveXMLNameRef(resolver, childScope, e.Name, e.GetPosition(), rootNeeds, true)
	for i := range e.Attrs {
		attr := &e.Attrs[i]
		resolveXMLNameRef(resolver, childScope, attr.Name, attr.GetPosition(), rootNeeds, false)
	}

	if e.Content != nil {
		resolveXMLContent(resolver, childScope, e.Content, rootNeeds)
	}
}

func resolveXMLContent[T symbolResolver](resolver T, scope model.Scope, content ast.BLangExpression, rootNeeds map[string]model.SymbolRef) {
	switch c := content.(type) {
	case *ast.BLangXMLElementLiteral:
		resolveXMLElementLiteralNamespaces(resolver, scope, c, rootNeeds)
	case *ast.BLangXMLSequenceLiteral:
		for _, child := range c.Children {
			resolveXMLContent(resolver, scope, child, rootNeeds)
		}
	}
}

func resolveXMLNameRef[T symbolResolver](resolver T, scope model.Scope, name string, pos diagnostics.Location, rootNeeds map[string]model.SymbolRef, isElement bool) model.SymbolRef {
	prefix, _ := splitXMLName(name)
	if prefix == "" {
		if !isElement {
			return model.SymbolRef{}
		}
		ref, defScope, ok := lookupXMLNS(scope, model.DefaultXMLNSSymbolName)
		if !ok || defScope == scope {
			return ref
		}
		if _, fromXMLAncestor := defScope.(*xmlnsChildScope); fromXMLAncestor {
			return ref
		}
		rootNeeds["xmlns"] = ref
		return ref
	}
	ref, defScope, ok := lookupXMLNS(scope, prefix)
	if !ok {
		semanticError(resolver, "undefined XML namespace prefix '"+prefix+"'", pos)
		return model.SymbolRef{}
	}
	if defScope != scope {
		if _, fromXMLAncestor := defScope.(*xmlnsChildScope); !fromXMLAncestor {
			rootNeeds["xmlns:"+prefix] = ref
		}
	}
	return ref
}

func stripInlineXMLNSAttrs[T symbolResolver](resolver T, childScope model.Scope, e *ast.BLangXMLElementLiteral) []ast.BLangXMLAttribute {
	kept := make([]ast.BLangXMLAttribute, 0, len(e.Attrs))
	for i := range e.Attrs {
		attr := e.Attrs[i]
		prefix, local := splitXMLName(attr.Name)
		if !isXMLNSAttr(prefix, local) {
			kept = append(kept, attr)
			continue
		}
		uri, ok := xmlnsAttrURI(resolver, &attr)
		if !ok {
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
			continue
		}
		e.Namespaces = append(e.Namespaces, ref)
	}
	return kept
}

func isXMLNSAttr(prefix, local string) bool {
	if prefix == "" {
		return local == "xmlns"
	}
	return prefix == "xmlns"
}

func xmlnsAttrURI[T symbolResolver](resolver T, attr *ast.BLangXMLAttribute) (string, bool) {
	if attr.Value == nil {
		semanticError(resolver, "xmlns attribute missing URI", attr.GetPosition())
		return "", false
	}
	lit, ok := attr.Value.(*ast.BLangLiteral)
	if !ok {
		semanticError(resolver, "xmlns attribute URI must be a string literal", attr.GetPosition())
		return "", false
	}
	uri, ok := lit.GetValue().(string)
	if !ok {
		semanticError(resolver, "xmlns attribute URI must be a string", attr.GetPosition())
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

func xmlnsDeclKey[T symbolResolver](resolver T, symbol model.Symbol) string {
	key, err := model.XMLNamespaceDeclKey(symbol)
	if err != nil {
		resolver.GetCtx().InternalError(err.Error(), diagnostics.Location{})
		return ""
	}
	return key
}

func mergeNamespaces[T symbolResolver](resolver T, root *ast.BLangXMLElementLiteral, extras map[string]model.SymbolRef) {
	existing := make(map[string]struct{}, len(root.Namespaces))
	for _, ref := range root.Namespaces {
		existing[xmlnsDeclKey(resolver, resolver.GetCtx().GetSymbol(ref))] = struct{}{}
	}
	for k, v := range extras {
		if _, exists := existing[k]; exists {
			continue
		}
		root.Namespaces = append(root.Namespaces, v)
		existing[k] = struct{}{}
	}
}

func appendXMLNSTemplateNamespace[T symbolResolver](resolver T, insn *ast.XMLTemplateNamespaceInsertion, seen map[string]struct{}, ref model.SymbolRef) {
	key := xmlnsDeclKey(resolver, resolver.GetCtx().GetSymbol(ref))
	if _, exists := seen[key]; exists {
		return
	}
	insn.Namespaces = append(insn.Namespaces, ref)
	seen[key] = struct{}{}
}

func resolveXMLTemplateNamespaces[T symbolResolver](resolver T, scope model.Scope, e *ast.BLangXMLTemplateExpr) {
	ensurePrefixMap(resolver, scope)
	for stringIndex := range e.NamespaceInsertions {
		for i := range e.NamespaceInsertions[stringIndex] {
			insn := &e.NamespaceInsertions[stringIndex][i]
			seen := make(map[string]struct{}, len(insn.Namespaces))
			for _, ref := range insn.Namespaces {
				seen[xmlnsDeclKey(resolver, resolver.GetCtx().GetSymbol(ref))] = struct{}{}
			}
			if insn.NeedsDefaultNS {
				if ref, _, ok := lookupXMLNS(scope, model.DefaultXMLNSSymbolName); ok {
					appendXMLNSTemplateNamespace(resolver, insn, seen, ref)
				}
			}
			for prefix := range insn.UsedPrefixes {
				ref, _, ok := lookupXMLNS(scope, prefix)
				if !ok {
					semanticError(resolver, "undefined XML namespace prefix '"+prefix+"'", e.GetPosition())
					continue
				}
				if prefix == "" || prefix == model.XMLNSReservedPrefix {
					continue
				}
				appendXMLNSTemplateNamespace(resolver, insn, seen, ref)
			}
		}
	}
}
