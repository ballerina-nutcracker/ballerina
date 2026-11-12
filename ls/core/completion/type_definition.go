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

	name                 string
	descriptor           []typeDefinitionToken // Tokens before the cursor.
	descriptorStart, end int
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
	ctx, ok := typeDefinitionContextAt(c)
	return ctx, ok
}

func buildTypeDefinition(c *cursor, match any) []protocol.CompletionItem {
	definitionContext, ok := match.(typeDefinitionContext)
	if !ok || definitionContext.position == typeDefinitionName {
		return emptyItems
	}
	if definitionContext.position == typeDefinitionQualifiedDescriptor {
		return typeDefinitionQualifiedItems(c, definitionContext.alias)
	}
	return typeDefinitionDescriptorItems(c, definitionContext)
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

// Only union aliases are expanded; other named types, and union aliases already
// being expanded on the current path (cycles), retain the "ref" category.
func flattenUnionLeaves(ctx *context.CompilerContext, pkgNode *ast.BLangPackage, descriptor ast.TypeDescriptor) []string {
	return flattenUnionLeavesOnPath(ctx, pkgNode, descriptor, map[*ast.BLangUnionTypeNode]bool{})
}

func flattenUnionLeavesOnPath(ctx *context.CompilerContext, pkgNode *ast.BLangPackage, descriptor ast.TypeDescriptor, expanding map[*ast.BLangUnionTypeNode]bool) []string {
	if union, ok := descriptor.(*ast.BLangUnionTypeNode); ok {
		leaves := flattenUnionLeavesOnPath(ctx, pkgNode, union.Lhs().TypeDescriptor, expanding)
		return append(leaves, flattenUnionLeavesOnPath(ctx, pkgNode, union.Rhs().TypeDescriptor, expanding)...)
	}
	if named, ok := descriptor.(*ast.BLangUserDefinedType); ok {
		if referent, ok := resolveNamedTypeDescriptor(ctx, pkgNode, named).(*ast.BLangUnionTypeNode); ok && !expanding[referent] {
			expanding[referent] = true
			defer delete(expanding, referent)
			return flattenUnionLeavesOnPath(ctx, pkgNode, referent, expanding)
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
	for _, imported := range typeDefinitionImports(c) {
		if imported.alias != alias {
			continue
		}
		projection, ok := c.sm.ImportedModuleProjection(imported.organization, imported.name)
		if ok {
			return projection.Symbols, projection.PackageNode, true
		}
	}
	return model.ExportedSymbolSpace{}, nil, false
}

type typeDefinitionImport struct {
	organization string
	name         string
	alias        string
}

func typeDefinitionImports(c *cursor) []typeDefinitionImport {
	name, ok := c.sm.NameOf(c.req.URI)
	if !ok {
		return nil
	}
	fileIndex := c.sm.Context().DiagnosticEnv().FileIndex(name)
	imports := make([]typeDefinitionImport, 0)
	for _, node := range c.sm.Imports() {
		position := node.GetPosition()
		if position.FileIndex() != fileIndex {
			continue
		}
		parts := make([]string, len(node.PkgNameComps))
		for index, part := range node.PkgNameComps {
			parts[index] = part.GetValue()
		}
		alias := parts[len(parts)-1]
		if node.Alias != nil {
			if node.Alias.OriginalValue == string(model.IGNORE) {
				continue
			}
			alias = node.Alias.GetValue()
		}
		organization := c.sm.PackageNode().PackageID.OrgName.Value()
		if node.OrgName != nil && node.OrgName.GetValue() != "" {
			organization = node.OrgName.GetValue()
		}
		imports = append(imports, typeDefinitionImport{organization: organization, name: strings.Join(parts, "."), alias: alias})
	}
	return imports
}

func typeDefinitionDescriptorItems(c *cursor, match typeDefinitionContext) []protocol.CompletionItem {
	set := newItemSet()
	if item, ok := recoveredTypeDefinitionItem(c, match); ok {
		set.add(item)
	}
	for _, item := range typeDefinitionScopeItems(c) {
		set.add(item)
	}
	for _, item := range typePositionLanguageItems(typeDefinitionHasTopLevelToken(match.descriptor, "|")) {
		set.add(item)
	}
	for _, item := range typeDefinitionPackageItems(c) {
		set.add(item)
	}
	for _, item := range typeDefinitionQualifiedPrefixItems(c, match.descriptor) {
		set.add(item)
	}
	items := set.items()
	return append(items, typePositionDescriptor("function", "Function", typeDefinitionOtherModuleOffset, typeDefinitionDefaultOffset))
}

func typeDefinitionQualifiedPrefixItems(c *cursor, descriptor []typeDefinitionToken) []protocol.CompletionItem {
	prefix := ""
	for i := len(descriptor) - 1; i >= 0; i-- {
		if descriptor[i].word && descriptor[i].depth == 0 {
			prefix = descriptor[i].text
			break
		}
	}
	if prefix == "" {
		return emptyItems
	}
	set := newItemSet()
	for _, imported := range typeDefinitionImports(c) {
		alias := imported.alias
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

func recoveredTypeDefinitionItem(c *cursor, match typeDefinitionContext) (protocol.CompletionItem, bool) {
	descriptor := typeDefinitionTokens(c.req.Text[match.descriptorStart:match.end])
	for i, token := range descriptor {
		if token.depth == 0 && token.text == ";" {
			descriptor = descriptor[:i]
			break
		}
	}
	nextWord := nextTopLevelWord(descriptor, 0)
	detail := ""
	primaryOffset, secondaryOffset := int8(0), int8(0)
	if nextWord < 0 || match.descriptorStart+descriptor[nextWord].start >= c.req.Offset {
		if nextWord >= 0 && descriptor[nextWord].text == "function" {
			detail, primaryOffset, secondaryOffset = "Function", typeDefinitionSameModuleOffset, typeDefinitionDefaultOffset
		}
	} else if typeDefinitionHasTopLevelToken(descriptor, "|") {
		detail, primaryOffset, secondaryOffset = "Union", typeDefinitionSameModuleOffset, typeDefinitionUnionOffset
	}
	if detail == "" {
		return protocol.CompletionItem{}, false
	}
	return protocol.CompletionItem{
		Label:      match.name,
		Kind:       protocol.NewOptional(protocol.CompletionItemKindTypeParameter),
		Detail:     protocol.NewOptional(detail),
		SortText:   protocol.NewOptional(typeDefinitionSortText(primaryOffset, secondaryOffset)),
		InsertText: protocol.NewOptional(match.name),
	}, true
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
	line := typeDefinitionImportInsertionLine(c)
	imports := typeDefinitionImports(c)
	for _, pkg := range c.packages {
		path := pkg.Organization + "/" + pkg.Name
		alias := pkg.Name
		if separator := strings.LastIndex(alias, "."); separator >= 0 {
			alias = alias[separator+1:]
		}
		if importedAlias, imported := typeDefinitionImportAlias(imports, pkg.Organization, pkg.Name); imported {
			alias = importedAlias
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

func typeDefinitionImportAlias(imports []typeDefinitionImport, organization, name string) (string, bool) {
	for _, imported := range imports {
		if imported.organization == organization && imported.name == name {
			return imported.alias, true
		}
	}
	return "", false
}

func typeDefinitionImportInsertionLine(c *cursor) int {
	name, ok := c.sm.NameOf(c.req.URI)
	if !ok {
		return 0
	}
	fileIndex := c.sm.Context().DiagnosticEnv().FileIndex(name)
	end := -1
	for _, node := range c.sm.Imports() {
		loc := node.GetPosition()
		if loc.FileIndex() == fileIndex && locationHasUsableOffsets(loc) && loc.EndOffset() <= len(c.req.Text) {
			end = max(end, loc.EndOffset())
		}
	}
	if end < 0 {
		return 0
	}
	return strings.Count(strings.TrimRight(c.req.Text[:end], " \t\r\n"), "\n") + 1
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

func typeDefinitionContextAt(c *cursor) (typeDefinitionContext, bool) {
	start, end, ok := typeDefinitionBounds(c)
	if !ok {
		return typeDefinitionContext{}, false
	}
	prefix := c.req.Text[start:c.req.Offset]
	tokens := typeDefinitionTokens(prefix)
	keyword := lastTypeDefinitionKeyword(tokens)
	if keyword < 0 || len(prefix) <= tokens[keyword].end {
		return typeDefinitionContext{}, false
	}
	name := nextTopLevelWord(tokens, keyword+1)
	if name < 0 || len(prefix) <= tokens[name].end {
		return typeDefinitionContext{position: typeDefinitionName}, true
	}
	descriptor := tokens[name+1:]
	defaultDepth := 0
	for _, token := range descriptor {
		if token.depth == 0 && token.text == ";" {
			return typeDefinitionContext{}, false
		}
		if defaultDepth > 0 && (token.depth < defaultDepth ||
			(token.depth == defaultDepth && (token.text == ";" || token.text == ","))) {
			defaultDepth = 0
		}
		if token.text == "=" && defaultDepth == 0 {
			defaultDepth = token.depth
		}
	}
	if defaultDepth > 0 {
		return typeDefinitionContext{}, false
	}
	if alias := qualifiedTypeAlias(prefix, descriptor); alias != "" {
		return typeDefinitionContext{position: typeDefinitionQualifiedDescriptor, alias: alias}, true
	}
	return typeDefinitionContext{
		position:        typeDefinitionDescriptor,
		name:            tokens[name].text,
		descriptor:      descriptor,
		descriptorStart: start + tokens[name].end,
		end:             end,
	}, true
}

func typeDefinitionBounds(c *cursor) (start, end int, ok bool) {
	if len(c.chain) > 1 {
		switch node := c.chain[1].(type) {
		case *ast.BLangTypeDefinition:
			if node.IsEnum() {
				return 0, 0, false
			}
			loc := node.GetPosition()
			if locationHasUsableOffsets(loc) && loc.StartOffset() <= c.req.Offset {
				return loc.StartOffset(), min(loc.EndOffset(), len(c.req.Text)), true
			}
		case *ast.BLangBadTopLevelNode:
		default:
			return 0, 0, false
		}
	}
	// A partly typed declaration may not have a type-definition node that covers the cursor yet.
	// Start at the current line, or at a recovered type definition that only whitespace separates from it.
	start = strings.LastIndexAny(c.req.Text[:c.req.Offset], "\r\n") + 1
	if name, ok := c.sm.NameOf(c.req.URI); ok {
		fileIndex := c.sm.Context().DiagnosticEnv().FileIndex(name)
		for _, definition := range c.sm.PackageNode().TypeDefinitions {
			loc := definition.GetPosition()
			if definition.IsEnum() || !locationHasUsableOffsets(loc) || loc.FileIndex() != fileIndex ||
				loc.StartOffset() >= start || loc.EndOffset() > start {
				continue
			}
			if strings.TrimSpace(c.req.Text[loc.EndOffset():start]) == "" {
				start = loc.StartOffset()
			}
		}
	}
	end = len(c.req.Text)
	if nextLine := strings.IndexAny(c.req.Text[c.req.Offset:], "\r\n"); nextLine >= 0 {
		end = c.req.Offset + nextLine
	}
	return start, end, true
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
		if (r == '}' || r == ']' || r == ')') && depth > 0 {
			depth--
		}
		tokens = append(tokens, typeDefinitionToken{text: text[offset : offset+size], start: offset, end: offset + size, depth: depth})
		if r == '{' || r == '[' || r == '(' {
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

func qualifiedTypeAlias(prefix string, tokens []typeDefinitionToken) string {
	if len(tokens) < 2 {
		return ""
	}
	last := tokens[len(tokens)-1]
	if strings.TrimSpace(prefix[last.end:]) != "" {
		return ""
	}
	if last.word {
		tokens = tokens[:len(tokens)-1]
	}
	if len(tokens) >= 2 && tokens[len(tokens)-1].text == ":" && tokens[len(tokens)-2].word {
		return tokens[len(tokens)-2].text
	}
	return ""
}
