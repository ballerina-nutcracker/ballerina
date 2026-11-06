// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package completion

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
	"github.com/ballerina-nutcracker/ballerina/model"
)

type typeDefinitionPosition uint8

const (
	typeDefinitionName typeDefinitionPosition = iota
	typeDefinitionDescriptor
	typeDefinitionQualifiedDescriptor
)

type typeDefinitionContext struct {
	position typeDefinitionPosition
	alias    string
}

const (
	typeDefinitionSameModuleOffset      int8 = 70
	typeDefinitionOtherModuleOffset     int8 = 60
	typeDefinitionAvailableModuleOffset int8 = 50
	typeDefinitionConstantOffset        int8 = 40
	typeDefinitionEnumOffset            int8 = 30
	typeDefinitionDescriptorOffset      int8 = 20
	typeDefinitionSnippetOffset         int8 = 10
	typeDefinitionKeywordOffset         int8 = 0

	typeDefinitionUnionOffset           int8 = 7
	typeDefinitionClassOffset           int8 = 5
	typeDefinitionErrorOffset           int8 = 4
	typeDefinitionRecordOffset          int8 = 3
	typeDefinitionDefaultOffset         int8 = 2
	typeDefinitionModuleOffset          int8 = 1
	typeDefinitionLangModuleOffset      int8 = 3
	typeDefinitionBallerinaModuleOffset int8 = 2
	typeDefinitionOtherPackageOffset    int8 = 1
)

func typeDefinitionSortText(primaryOffset, secondaryOffset int8) string {
	return neutralRelevance.add(primaryOffset).add(secondaryOffset).sortText()
}

var typeDefinitionHandler = handler{
	syntaxOnly: true,
	name:       "type-definition",
	match:      matchTypeDefinition,
	build:      buildTypeDefinition,
}

func matchTypeDefinition(c *cursor) (any, bool) {
	ctx, ok := typeDefinitionContextAt(c.req)
	return ctx, ok
}

func buildTypeDefinition(c *cursor, context any) []protocol.CompletionItem {
	definitionContext, ok := context.(typeDefinitionContext)
	if !ok || definitionContext.position == typeDefinitionName {
		return emptyItems
	}
	if definitionContext.position == typeDefinitionQualifiedDescriptor {
		return typeDefinitionQualifiedItems(c, definitionContext.alias)
	}
	return typeDefinitionDescriptorItems(c)
}

func typeDefinitionQualifiedItems(c *cursor, alias string) []protocol.CompletionItem {
	contents, pkgNode, ok := typeDefinitionModuleContents(c, alias)
	if !ok {
		return emptyItems
	}
	ctx := c.sm.Context()
	members := enumMemberNames(ctx, pkgNode)
	set := newItemSet()
	for ref := range contents.PublicMainSymbols() {
		kind := ctx.SymbolKind(ref)
		if kind != model.SymbolKindType && kind != model.SymbolKindConstant {
			continue
		}
		set.add(typeDefinitionSymbolItem(c, pkgNode, ref, members[ctx.SymbolName(ref)]))
	}
	return set.items()
}

func typeDefinitionSymbolItem(c *cursor, pkgNode *ast.BLangPackage, ref model.SymbolRef, enumMember bool) protocol.CompletionItem {
	ctx := c.sm.Context()
	name := ctx.SymbolName(ref)
	kind := ctx.SymbolKind(ref)
	item := protocol.CompletionItem{
		Label:      name,
		Kind:       protocol.NewOptional(completionItemKind(kind)),
		InsertText: protocol.NewOptional(name),
	}
	if kind == model.SymbolKindConstant {
		return typeDefinitionConstantItem(c, pkgNode, ref, name, item, enumMember)
	}
	shape := classifyTypeSymbol(c, pkgNode, ref)
	if shape.detail != "" {
		item.Detail = protocol.NewOptional(shape.detail)
	}
	item.SortText = protocol.NewOptional(typeDefinitionSortText(shape.primaryOffset, shape.secondaryOffset))
	return item
}

func typeDefinitionConstantItem(c *cursor, pkgNode *ast.BLangPackage, ref model.SymbolRef, name string, item protocol.CompletionItem, enumMember bool) protocol.CompletionItem {
	switch {
	case typeDefinitionCurrentDocumentSymbol(c, ref):
		item.Detail = protocol.NewOptional("Singleton")
	default:
		if detail := symbolDetail(c, ref); detail != "" {
			item.Detail = protocol.NewOptional(detail)
		} else if literal := constantLiteralDetail(pkgNode, name); literal != "" {
			item.Detail = protocol.NewOptional(literal)
		}
	}
	if enumMember {
		item.SortText = protocol.NewOptional(typeDefinitionSortText(typeDefinitionEnumOffset, 0))
	} else {
		item.SortText = protocol.NewOptional(typeDefinitionSortText(typeDefinitionConstantOffset, 0))
	}
	return item
}

func constantLiteralDetail(pkgNode *ast.BLangPackage, name string) string {
	if pkgNode == nil {
		return ""
	}
	for _, constant := range pkgNode.Constants {
		if constant.Name.GetValue() != name {
			continue
		}
		literal, ok := constant.Expr.(*ast.BLangLiteral)
		if !ok {
			return ""
		}
		value := literal.GetOriginalValue()
		if literal.GetLiteralKind() == ast.LiteralKindString && !strings.HasPrefix(value, "\"") {
			// Implicit enum-member string values lack source quotes.
			return "\"" + value + "\""
		}
		return value
	}
	return ""
}

type typeShape struct {
	detail          string
	primaryOffset   int8
	secondaryOffset int8
}

func classifyTypeSymbol(c *cursor, pkgNode *ast.BLangPackage, ref model.SymbolRef) typeShape {
	ctx := c.sm.Context()
	if ctx.SymbolIsClass(ref) {
		return typeShape{detail: "Class", primaryOffset: typeDefinitionOtherModuleOffset, secondaryOffset: typeDefinitionClassOffset}
	}
	descriptor, owner := findTypeDescriptorAnywhere(c, pkgNode, ctx.SymbolName(ref))
	if descriptor == nil {
		return classifySymbolFallback(ctx, ref)
	}
	if isEnumDescriptor(ctx, descriptor) {
		return typeShape{detail: "enum", primaryOffset: typeDefinitionEnumOffset}
	}
	return classifyTypeDescriptor(ctx, owner, descriptor)
}

func findTypeDescriptorAnywhere(c *cursor, pkgNode *ast.BLangPackage, name string) (ast.TypeDescriptor, *ast.BLangPackage) {
	if descriptor := findTypeDescriptor(pkgNode, name); descriptor != nil {
		return descriptor, pkgNode
	}
	for _, pkg := range c.packages {
		projection, ok := c.sm.ExternalModuleProjection(pkg.Organization, pkg.Name)
		if !ok {
			continue
		}
		if descriptor := findTypeDescriptor(projection.PackageNode, name); descriptor != nil {
			return descriptor, projection.PackageNode
		}
	}
	return nil, nil
}

func classifySymbolFallback(ctx *context.CompilerContext, ref model.SymbolRef) typeShape {
	switch ctx.GetSymbol(ref).(type) {
	case model.ClassSymbol:
		return typeShape{detail: "Class", primaryOffset: typeDefinitionOtherModuleOffset, secondaryOffset: typeDefinitionClassOffset}
	case *model.RecordSymbol:
		return typeShape{detail: "Record", primaryOffset: typeDefinitionOtherModuleOffset, secondaryOffset: typeDefinitionRecordOffset}
	case *model.ObjectTypeSymbol:
		return typeShape{detail: "Object", primaryOffset: typeDefinitionOtherModuleOffset, secondaryOffset: typeDefinitionClassOffset}
	case *model.ErrorTypeSymbol:
		return typeShape{detail: "Error", primaryOffset: typeDefinitionOtherModuleOffset, secondaryOffset: typeDefinitionErrorOffset}
	default:
		return typeShape{primaryOffset: typeDefinitionOtherModuleOffset, secondaryOffset: typeDefinitionDefaultOffset}
	}
}

func classifyTypeDescriptor(ctx *context.CompilerContext, pkgNode *ast.BLangPackage, descriptor ast.TypeDescriptor) typeShape {
	switch node := descriptor.(type) {
	case *ast.BLangRecordType:
		return typeShape{detail: "Record", primaryOffset: typeDefinitionOtherModuleOffset, secondaryOffset: typeDefinitionRecordOffset}
	case *ast.BLangObjectType:
		return typeShape{detail: "Object", primaryOffset: typeDefinitionOtherModuleOffset, secondaryOffset: typeDefinitionClassOffset}
	case *ast.BLangErrorTypeNode:
		return typeShape{detail: "Error", primaryOffset: typeDefinitionOtherModuleOffset, secondaryOffset: typeDefinitionErrorOffset}
	case *ast.BLangUnionTypeNode:
		return classifyUnion(ctx, pkgNode, node)
	case *ast.BLangConstrainedType:
		return classifyConstrained(node)
	case *ast.BLangValueType:
		return typeShape{detail: capitalizeTypeKind(node.TypeKind), primaryOffset: typeDefinitionOtherModuleOffset, secondaryOffset: typeDefinitionDefaultOffset}
	case *ast.BLangBuiltInRefTypeNode:
		return typeShape{detail: capitalizeTypeKind(node.TypeKind), primaryOffset: typeDefinitionOtherModuleOffset, secondaryOffset: typeDefinitionDefaultOffset}
	case *ast.BLangFiniteTypeNode:
		return typeShape{detail: "Finite", primaryOffset: typeDefinitionOtherModuleOffset, secondaryOffset: typeDefinitionDefaultOffset}
	default:
		return typeShape{primaryOffset: typeDefinitionOtherModuleOffset, secondaryOffset: typeDefinitionDefaultOffset}
	}
}

func capitalizeTypeKind(kind ast.TypeKind) string {
	text := kind.String()
	if text == "" {
		return text
	}
	return strings.ToUpper(text[:1]) + strings.ToLower(text[1:])
}

func classifyUnion(ctx *context.CompilerContext, pkgNode *ast.BLangPackage, union *ast.BLangUnionTypeNode) typeShape {
	leaves := flattenUnionLeaves(ctx, pkgNode, union)
	secondaryOffset := typeDefinitionDefaultOffset
	homogeneous := len(leaves) > 0
	for _, leaf := range leaves[1:] {
		if leaf != leaves[0] {
			homogeneous = false
			break
		}
	}
	switch {
	case !homogeneous:
		secondaryOffset = typeDefinitionUnionOffset
	case leaves[0] == "error":
		secondaryOffset = typeDefinitionErrorOffset
	case leaves[0] == "record":
		secondaryOffset = typeDefinitionRecordOffset
	case leaves[0] == "object":
		secondaryOffset = typeDefinitionClassOffset
	}
	return typeShape{detail: "Union", primaryOffset: typeDefinitionOtherModuleOffset, secondaryOffset: secondaryOffset}
}

// Only union aliases are expanded; other named types retain the "ref" category.
func flattenUnionLeaves(ctx *context.CompilerContext, pkgNode *ast.BLangPackage, descriptor ast.TypeDescriptor) []string {
	if union, ok := descriptor.(*ast.BLangUnionTypeNode); ok {
		leaves := flattenUnionLeaves(ctx, pkgNode, union.Lhs().TypeDescriptor)
		return append(leaves, flattenUnionLeaves(ctx, pkgNode, union.Rhs().TypeDescriptor)...)
	}
	if named, ok := descriptor.(*ast.BLangUserDefinedType); ok {
		if referent := resolveNamedTypeDescriptor(ctx, pkgNode, named); referent != nil {
			if _, ok := referent.(*ast.BLangUnionTypeNode); ok {
				return flattenUnionLeaves(ctx, pkgNode, referent)
			}
		}
	}
	return []string{unionLeafCategory(descriptor)}
}

// Named-type lookup is limited to pkgNode.
func resolveNamedTypeDescriptor(ctx *context.CompilerContext, pkgNode *ast.BLangPackage, named *ast.BLangUserDefinedType) ast.TypeDescriptor {
	ref := named.Symbol()
	if ref.IsEmpty() || ctx.SymbolKind(ref) != model.SymbolKindType {
		return nil
	}
	return findTypeDescriptor(pkgNode, ctx.SymbolName(ref))
}

// Keep basic types distinct so int|string is heterogeneous.
func unionLeafCategory(descriptor ast.TypeDescriptor) string {
	switch node := descriptor.(type) {
	case *ast.BLangRecordType:
		return "record"
	case *ast.BLangObjectType:
		return "object"
	case *ast.BLangErrorTypeNode:
		return "error"
	case *ast.BLangUserDefinedType:
		return "ref"
	case *ast.BLangValueType:
		return "value:" + node.TypeKind.String()
	case *ast.BLangBuiltInRefTypeNode:
		return "value:" + node.TypeKind.String()
	case *ast.BLangArrayType:
		return "array"
	case *ast.BLangFiniteTypeNode:
		return "finite"
	case *ast.BLangFunctionType:
		return "function"
	case *ast.BLangConstrainedType:
		return "constrained:" + fmt.Sprint(safeConstraintKind(node))
	default:
		return fmt.Sprintf("other:%T", descriptor)
	}
}

func classifyConstrained(node *ast.BLangConstrainedType) typeShape {
	switch safeConstraintKind(node) {
	case ast.TypeKindTypeDesc:
		return typeShape{detail: "Typedesc", primaryOffset: typeDefinitionOtherModuleOffset, secondaryOffset: typeDefinitionDefaultOffset}
	case ast.TypeKindMap:
		return typeShape{detail: "Map", primaryOffset: typeDefinitionOtherModuleOffset, secondaryOffset: typeDefinitionDefaultOffset}
	default:
		return typeShape{primaryOffset: typeDefinitionOtherModuleOffset, secondaryOffset: typeDefinitionDefaultOffset}
	}
}

func safeConstraintKind(node *ast.BLangConstrainedType) ast.TypeKind {
	switch base := node.GetType().TypeDescriptor.(type) {
	case *ast.BLangBuiltInRefTypeNode:
		return base.TypeKind
	case *ast.BLangValueType:
		return base.TypeKind
	default:
		return ast.TypeKindNone
	}
}

// Enum descriptors consist of constant references, possibly joined by unions.
func isEnumDescriptor(ctx *context.CompilerContext, descriptor ast.TypeDescriptor) bool {
	refs, ok := enumConstantRefs(ctx, descriptor)
	return ok && len(refs) > 0
}

func enumMemberNames(ctx *context.CompilerContext, pkgNode *ast.BLangPackage) map[string]bool {
	members := make(map[string]bool)
	if pkgNode == nil {
		return members
	}
	for _, definition := range pkgNode.TypeDefinitions {
		refs, ok := enumConstantRefs(ctx, definition.GetTypeData().TypeDescriptor)
		if !ok {
			continue
		}
		for _, ref := range refs {
			members[ctx.SymbolName(ref)] = true
		}
	}
	return members
}

func enumConstantRefs(ctx *context.CompilerContext, descriptor ast.TypeDescriptor) ([]model.SymbolRef, bool) {
	if union, ok := descriptor.(*ast.BLangUnionTypeNode); ok {
		left, ok := enumConstantRefs(ctx, union.Lhs().TypeDescriptor)
		if !ok {
			return nil, false
		}
		right, ok := enumConstantRefs(ctx, union.Rhs().TypeDescriptor)
		if !ok {
			return nil, false
		}
		return append(left, right...), true
	}
	named, ok := descriptor.(*ast.BLangUserDefinedType)
	if !ok {
		return nil, false
	}
	ref := named.Symbol()
	if ref.IsEmpty() || ctx.SymbolKind(ref) != model.SymbolKindConstant {
		return nil, false
	}
	return []model.SymbolRef{ref}, true
}

func findTypeDescriptor(pkgNode *ast.BLangPackage, name string) ast.TypeDescriptor {
	if pkgNode == nil {
		return nil
	}
	for _, definition := range pkgNode.TypeDefinitions {
		if definition.Name.GetValue() == name {
			return definition.GetTypeData().TypeDescriptor
		}
	}
	return nil
}

func typeDefinitionModuleContents(c *cursor, alias string) (model.ExportedSymbolSpace, *ast.BLangPackage, bool) {
	if contents, ok := moduleContentSymbols(nearestScope(c.chain), alias); ok {
		return contents, c.sm.PackageNode(), true
	}
	if contents, ok := moduleContentSymbols(c.sm.PackageNode().Scope, alias); ok {
		return contents, c.sm.PackageNode(), true
	}
	finder := &typeDefinitionModuleContentFinder{alias: alias}
	ast.Walk(finder, c.sm.PackageNode())
	if finder.found {
		return finder.contents, c.sm.PackageNode(), true
	}
	for _, pkg := range c.packages {
		packageAlias := pkg.Name
		if separator := strings.LastIndex(packageAlias, "."); separator >= 0 {
			packageAlias = packageAlias[separator+1:]
		}
		if packageAlias != alias || !typeDefinitionHasImport(c.req.Text, pkg.Organization+"/"+pkg.Name) {
			continue
		}
		projection, ok := c.sm.ExternalModuleProjection(pkg.Organization, pkg.Name)
		if ok {
			return projection.Symbols, projection.PackageNode, true
		}
	}
	return model.ExportedSymbolSpace{}, nil, false
}

type typeDefinitionModuleContentFinder struct {
	alias    string
	contents model.ExportedSymbolSpace
	found    bool
}

func (f *typeDefinitionModuleContentFinder) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil || f.found {
		return f
	}
	scoped, ok := node.(ast.NodeWithScope)
	if !ok {
		return f
	}
	contents, ok := moduleContentSymbols(scoped.Scope(), f.alias)
	if !ok {
		return f
	}
	f.contents = contents
	f.found = true
	return f
}

func (f *typeDefinitionModuleContentFinder) VisitTypeData(*ast.TypeData) ast.Visitor {
	return f
}

func typeDefinitionDescriptorItems(c *cursor) []protocol.CompletionItem {
	set := newItemSet()
	if item, ok := recoveredTypeDefinitionItem(c); ok {
		set.add(item)
	}
	for _, item := range typeDefinitionScopeItems(c) {
		set.add(item)
	}
	for _, item := range typePositionLanguageItems(typeDefinitionUnionContinuation(c.req)) {
		set.add(item)
	}
	for _, item := range typeDefinitionPackageItems(c) {
		set.add(item)
	}
	for _, item := range typeDefinitionQualifiedPrefixItems(c) {
		set.add(item)
	}
	items := set.items()
	return append(items, typePositionDescriptor("function", "Function", typeDefinitionOtherModuleOffset, typeDefinitionDefaultOffset))
}

func typeDefinitionQualifiedPrefixItems(c *cursor) []protocol.CompletionItem {
	prefix := typeDefinitionDescriptorPrefix(c.req)
	if prefix == "" {
		return emptyItems
	}
	set := newItemSet()
	for _, pkg := range c.packages {
		path := pkg.Organization + "/" + pkg.Name
		if !typeDefinitionHasImport(c.req.Text, path) {
			continue
		}
		alias := pkg.Name
		if separator := strings.LastIndex(alias, "."); separator >= 0 {
			alias = alias[separator+1:]
		}
		contents, pkgNode, ok := typeDefinitionModuleContents(c, alias)
		if !ok {
			continue
		}
		for ref := range contents.PublicMainSymbols() {
			if c.sm.Context().SymbolKind(ref) != model.SymbolKindType {
				continue
			}
			name := c.sm.Context().SymbolName(ref)
			if !strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
				continue
			}
			item := typeDefinitionSymbolItem(c, pkgNode, ref, false)
			item.Label = alias + ":" + name
			item.InsertText = protocol.NewOptional(item.Label)
			set.add(item)
		}
	}
	return set.items()
}

func typeDefinitionDescriptorPrefix(req Request) string {
	tokens := typeDefinitionTokens(req.Text[:req.Offset])
	keyword := lastTypeDefinitionKeyword(tokens)
	if keyword < 0 {
		return ""
	}
	name := nextTopLevelWord(tokens, keyword+1)
	if name < 0 {
		return ""
	}
	for index := len(tokens) - 1; index > name; index-- {
		if tokens[index].word && tokens[index].depth == 0 {
			return tokens[index].text
		}
	}
	return ""
}

func recoveredTypeDefinitionItem(c *cursor) (protocol.CompletionItem, bool) {
	tokens := typeDefinitionTokens(c.req.Text)
	keyword := lastTypeDefinitionKeyword(tokens)
	if keyword < 0 {
		return protocol.CompletionItem{}, false
	}
	name := nextTopLevelWord(tokens, keyword+1)
	if name < 0 {
		return protocol.CompletionItem{}, false
	}
	descriptor := tokens[name+1:]
	nextWord := nextTopLevelWord(tokens, name+1)
	detail := ""
	primaryOffset, secondaryOffset := int8(0), int8(0)
	if nextWord < 0 || tokens[nextWord].start >= c.req.Offset {
		if typeDefinitionNextWord(tokens, name+1) == "function" {
			detail, primaryOffset, secondaryOffset = "Function", typeDefinitionSameModuleOffset, typeDefinitionDefaultOffset
		}
	} else if typeDefinitionHasTopLevelToken(descriptor, "|") {
		detail, primaryOffset, secondaryOffset = "Union", typeDefinitionSameModuleOffset, typeDefinitionUnionOffset
	}
	if detail == "" {
		return protocol.CompletionItem{}, false
	}
	return protocol.CompletionItem{
		Label:      tokens[name].text,
		Kind:       protocol.NewOptional(protocol.CompletionItemKindTypeParameter),
		Detail:     protocol.NewOptional(detail),
		SortText:   protocol.NewOptional(typeDefinitionSortText(primaryOffset, secondaryOffset)),
		InsertText: protocol.NewOptional(tokens[name].text),
	}, true
}

func typeDefinitionNextWord(tokens []typeDefinitionToken, start int) string {
	for index := start; index < len(tokens); index++ {
		if tokens[index].word && tokens[index].depth == 0 {
			return tokens[index].text
		}
	}
	return ""
}

func typeDefinitionHasTopLevelToken(tokens []typeDefinitionToken, text string) bool {
	for _, token := range tokens {
		if token.depth == 0 && token.text == text {
			return true
		}
	}
	return false
}

func typeDefinitionCurrentDocumentSymbol(c *cursor, ref model.SymbolRef) bool {
	location := c.sm.Context().SymbolLocation(ref)
	name, ok := c.sm.NameOf(c.req.URI)
	return ok && c.sm.Context().DiagnosticEnv().FileName(location) == name
}

func typeDefinitionPackageItems(c *cursor) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(c.packages))
	line := typeDefinitionImportInsertionLine(c.req.Text)
	for _, pkg := range c.packages {
		path := pkg.Organization + "/" + pkg.Name
		alias := pkg.Name
		if separator := strings.LastIndex(alias, "."); separator >= 0 {
			alias = alias[separator+1:]
		}
		if typeDefinitionHasImport(c.req.Text, path) {
			items = append(items, protocol.CompletionItem{
				Label:      alias,
				Kind:       protocol.NewOptional(protocol.CompletionItemKindModule),
				Detail:     protocol.NewOptional("Module"),
				SortText:   protocol.NewOptional(typeDefinitionSortText(typeDefinitionOtherModuleOffset, typeDefinitionModuleOffset)),
				FilterText: protocol.NewOptional(alias),
				InsertText: protocol.NewOptional(alias),
			})
			continue
		}
		primaryOffset, secondaryOffset := typeDefinitionPackageOffsets(pkg)
		items = append(items, protocol.CompletionItem{
			Label:      path,
			Kind:       protocol.NewOptional(protocol.CompletionItemKindModule),
			Detail:     protocol.NewOptional("Module"),
			SortText:   protocol.NewOptional(typeDefinitionSortText(primaryOffset, secondaryOffset)),
			FilterText: protocol.NewOptional(alias),
			InsertText: protocol.NewOptional(alias),
			AdditionalTextEdits: protocol.NewOptional([]protocol.TextEdit{{
				Range:   protocol.Range{Start: protocol.Position{Line: uint32(line)}, End: protocol.Position{Line: uint32(line)}},
				NewText: "import " + path + ";\n",
			}}),
		})
	}
	return items
}

func typeDefinitionPackageOffsets(pkg AvailablePackage) (int8, int8) {
	if pkg.Organization == "ballerina" {
		if strings.HasPrefix(pkg.Name, "lang.") {
			return typeDefinitionAvailableModuleOffset, typeDefinitionLangModuleOffset
		}
		return typeDefinitionAvailableModuleOffset, typeDefinitionBallerinaModuleOffset
	}
	return typeDefinitionAvailableModuleOffset, typeDefinitionOtherPackageOffset
}

func typeDefinitionHasImport(text, packagePath string) bool {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "import ") && strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(line, "import ")), ";") == packagePath {
			return true
		}
	}
	return false
}

func typeDefinitionImportInsertionLine(text string) int {
	line := 0
	for _, value := range strings.Split(text, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(value), "import ") {
			return line
		}
		line++
	}
	return line
}

func typeDefinitionScopeItems(c *cursor) []protocol.CompletionItem {
	packageScope, ok := c.sm.PackageNode().Scope.(*model.PackageScope)
	if !ok {
		return emptyItems
	}
	spaces := append([]*model.SymbolSpace{}, packageScope.MainSpaces...)
	if packageScope.Virtual != nil {
		spaces = append(spaces, packageScope.Virtual.Main)
	}
	ctx := c.sm.Context()
	pkgNode := c.sm.PackageNode()
	members := enumMemberNames(ctx, pkgNode)
	seen := make(map[model.SymbolRef]bool)
	items := make([]protocol.CompletionItem, 0)
	for _, space := range spaces {
		if space == nil {
			continue
		}
		for ref := range space.Symbols() {
			if seen[ref] {
				continue
			}
			seen[ref] = true
			kind := ctx.SymbolKind(ref)
			if kind != model.SymbolKindType && kind != model.SymbolKindConstant {
				continue
			}
			items = append(items, typeDefinitionSymbolItem(c, pkgNode, ref, members[ctx.SymbolName(ref)]))
		}
	}
	return items
}

func typeDefinitionUnionContinuation(req Request) bool {
	tokens := typeDefinitionTokens(req.Text[:req.Offset])
	keyword := lastTypeDefinitionKeyword(tokens)
	if keyword < 0 {
		return false
	}
	name := nextTopLevelWord(tokens, keyword+1)
	return name >= 0 && typeDefinitionHasTopLevelToken(tokens[name+1:], "|")
}

func typePositionLanguageItems(unionContinuation bool) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, 36)
	keywordSortText := typeDefinitionSortText(typeDefinitionKeywordOffset, 0)
	keywords := []snippetID{kwRecord, kwFunction, kwDistinct, kwTrue, kwFalse, kwIsolated, kwClient, kwTransactional, kwNil, kwService}
	for _, id := range keywords {
		if unionContinuation && (id == kwIsolated || id == kwClient || id == kwTransactional) {
			continue
		}
		items = append(items, snippetItem(id, keywordSortText))
	}
	snippetSortText := typeDefinitionSortText(typeDefinitionSnippetOffset, 0)
	for _, id := range []snippetID{defRecordTypeDesc, defClosedRecordTypeDesc, defObjectTypeDescSnippet} {
		items = append(items, snippetItem(id, snippetSortText))
	}
	for _, label := range []string{"readonly", "handle", "never", "json", "anydata", "any", "byte", "decimal", "xml", "boolean", "future", "int", "float", "function", "string", "typedesc"} {
		detail := strings.ToUpper(label[:1]) + label[1:]
		items = append(items, typePositionDescriptor(label, detail, typeDefinitionOtherModuleOffset, typeDefinitionDefaultOffset))
	}
	items = append(items, typePositionDescriptor("error", "Error", typeDefinitionOtherModuleOffset, typeDefinitionErrorOffset))
	for _, label := range []string{"map", "object", "stream", "table", "transaction", "natural"} {
		items = append(items, typePositionDescriptor(label, "type", typeDefinitionDescriptorOffset, 0))
	}
	return items
}

func typePositionDescriptor(label, detail string, primaryOffset, secondaryOffset int8) protocol.CompletionItem {
	return protocol.CompletionItem{
		Label:      label,
		Kind:       protocol.NewOptional(protocol.CompletionItemKindTypeParameter),
		Detail:     protocol.NewOptional(detail),
		SortText:   protocol.NewOptional(typeDefinitionSortText(primaryOffset, secondaryOffset)),
		InsertText: protocol.NewOptional(label),
	}
}

func typeDefinitionContextAt(req Request) (typeDefinitionContext, bool) {
	if req.Offset < 0 || req.Offset > len(req.Text) {
		return typeDefinitionContext{}, false
	}
	tokens := typeDefinitionTokens(req.Text[:req.Offset])
	keyword := lastTypeDefinitionKeyword(tokens)
	if keyword < 0 {
		return typeDefinitionContext{}, false
	}
	name := nextTopLevelWord(tokens, keyword+1)
	if name < 0 || req.Offset <= tokens[name].end {
		return typeDefinitionContext{position: typeDefinitionName}, true
	}
	for _, token := range tokens[name+1:] {
		if token.depth == 0 && token.text == ";" {
			return typeDefinitionContext{}, false
		}
	}
	if alias := qualifiedTypeAlias(tokens, name+1); alias != "" {
		return typeDefinitionContext{position: typeDefinitionQualifiedDescriptor, alias: alias}, true
	}
	return typeDefinitionContext{position: typeDefinitionDescriptor}, true
}

type typeDefinitionToken struct {
	text       string
	start, end int
	depth      int
	word       bool
}

func typeDefinitionTokens(text string) []typeDefinitionToken {
	tokens := make([]typeDefinitionToken, 0)
	depth := 0
	for offset := 0; offset < len(text); {
		switch text[offset] {
		case '/':
			if offset+1 < len(text) && text[offset+1] == '/' {
				offset = skipToNewline(text, offset+2)
				continue
			}
			if offset+1 < len(text) && text[offset+1] == '*' {
				offset = skipBlockComment(text, offset+2)
				continue
			}
		case '#':
			offset = skipToNewline(text, offset+1)
			continue
		case '"', '`':
			offset = skipQuotedText(text, offset, text[offset])
			continue
		}
		r, size := utf8.DecodeRuneInString(text[offset:])
		if isTypeDefinitionWordRune(r) {
			start := offset
			offset += size
			for offset < len(text) {
				r, size = utf8.DecodeRuneInString(text[offset:])
				if !isTypeDefinitionWordRune(r) {
					break
				}
				offset += size
			}
			tokens = append(tokens, typeDefinitionToken{text: text[start:offset], start: start, end: offset, depth: depth, word: true})
			continue
		}
		if r == '}' && depth > 0 {
			depth--
		}
		tokens = append(tokens, typeDefinitionToken{text: text[offset : offset+size], start: offset, end: offset + size, depth: depth})
		if r == '{' {
			depth++
		}
		offset += size
	}
	return tokens
}

func isTypeDefinitionWordRune(r rune) bool {
	return r == '_' || r == '\'' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func skipToNewline(text string, offset int) int {
	for offset < len(text) && text[offset] != '\n' && text[offset] != '\r' {
		offset++
	}
	return offset
}

func skipBlockComment(text string, offset int) int {
	for offset+1 < len(text) {
		if text[offset] == '*' && text[offset+1] == '/' {
			return offset + 2
		}
		offset++
	}
	return len(text)
}

func skipQuotedText(text string, offset int, quote byte) int {
	offset++
	for offset < len(text) {
		if text[offset] == '\\' && quote == '"' && offset+1 < len(text) {
			offset += 2
			continue
		}
		if text[offset] == quote {
			return offset + 1
		}
		offset++
	}
	return len(text)
}

func lastTypeDefinitionKeyword(tokens []typeDefinitionToken) int {
	for index := len(tokens) - 1; index >= 0; index-- {
		token := tokens[index]
		if token.depth == 0 && token.word && token.text == "type" {
			return index
		}
	}
	return -1
}

func nextTopLevelWord(tokens []typeDefinitionToken, start int) int {
	for index := start; index < len(tokens); index++ {
		if tokens[index].depth == 0 && tokens[index].word {
			return index
		}
	}
	return -1
}

func qualifiedTypeAlias(tokens []typeDefinitionToken, start int) string {
	segmentStart := start
	for index := start; index < len(tokens); index++ {
		token := tokens[index]
		if token.depth != 0 {
			continue
		}
		if token.text == "|" {
			segmentStart = index + 1
		}
	}
	for index := segmentStart; index+1 < len(tokens); index++ {
		if tokens[index].depth != 0 || !tokens[index].word || tokens[index+1].depth != 0 || tokens[index+1].text != ":" {
			continue
		}
		return tokens[index].text
	}
	return ""
}
