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

package bir

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

type PrettyPrinter struct {
	indentLevel int
	sb          strings.Builder
	cx          semtypes.Context
}

// writeLine writes a line with current indentation and newline
func (p *PrettyPrinter) writeLine(s string) {
	for i := 0; i < p.indentLevel; i++ {
		p.sb.WriteString("  ")
	}
	p.sb.WriteString(s)
	p.sb.WriteString("\n")
}

// write writes without indentation or newline
func (p *PrettyPrinter) write(s string) {
	p.sb.WriteString(s)
}

// writeIndent writes current indentation without content or newline
func (p *PrettyPrinter) writeIndent() {
	for i := 0; i < p.indentLevel; i++ {
		p.sb.WriteString("  ")
	}
}

// increaseIndent increases indentation level
func (p *PrettyPrinter) increaseIndent() {
	p.indentLevel++
}

// decreaseIndent decreases indentation level
func (p *PrettyPrinter) decreaseIndent() {
	p.indentLevel--
}

func (p *PrettyPrinter) Print(tyCtx semtypes.Context, node BIRPackage) string {
	// Reset the builder
	p.sb.Reset()
	p.cx = tyCtx

	p.write("module ")
	p.write(p.PrintPackageID(node.PackageID))
	p.write(";\n")
	sortedGlobalVars := make([]BIRGlobalVariableDcl, 0, len(node.GlobalVars))
	for _, globalVar := range node.GlobalVars {
		sortedGlobalVars = append(sortedGlobalVars, globalVar)
	}
	sort.Slice(sortedGlobalVars, func(i, j int) bool {
		return string(sortedGlobalVars[i].GetName()) < string(sortedGlobalVars[j].GetName())
	})
	for _, globalVar := range sortedGlobalVars {
		p.write(p.PrintGlobalVar(globalVar))
		p.write(";\n")
	}
	for _, classDef := range node.ClassDefs {
		p.PrintClassDef(classDef)
		p.write("\n")
	}
	for _, function := range node.Functions {
		p.PrintFunction(function)
		p.write("\n")
	}
	return p.sb.String()
}

// printFunctionParams prints the parameter list of function. Native dependently
// typed functions carry no local variables because their signature is only known
// at each call site, so there is nothing to print for them.
func (p *PrettyPrinter) printFunctionParams(function BIRFunction) {
	paramStart := function.ParamLocalVarOffset()
	if len(function.LocalVars) <= paramStart {
		return
	}
	for i, v := range function.LocalVars[paramStart:] {
		if i >= len(function.RequiredParams) {
			break
		}
		if i > 0 {
			p.write(",")
		}
		p.printAnnotations(function.RequiredParams[i].Annotations)
		p.write(p.PrintSemType(v.Type))
	}
	if function.RestParams != nil {
		variableIndex := paramStart + len(function.RequiredParams)
		if variableIndex != paramStart {
			p.write(",")
		}
		p.printAnnotations(function.RestParams.Annotations)
		p.write(p.PrintSemType(function.LocalVars[variableIndex].Type))
		p.write("...")
	}
}

func (p *PrettyPrinter) PrintFunction(function BIRFunction) {
	p.write(function.Name.Value())
	p.write("(")
	p.printFunctionParams(function)
	p.write(")")
	if function.ReturnVariable != nil && !semtypes.IsZero(function.ReturnVariable.Type) {
		p.write(" -> ")
		p.write(p.PrintSemType(function.ReturnVariable.Type))
	}
	p.write("{\n")
	p.increaseIndent()
	for _, basicBlock := range function.BasicBlocks {
		p.PrintBasicBlock(basicBlock)
	}
	if len(function.ErrorTable) > 0 {
		p.writeLine("")
		p.writeLine("error-table {")
		p.increaseIndent()
		for _, entry := range function.ErrorTable {
			p.writeLine(fmt.Sprintf("[bb%d, bb%d] -> bb%d, %s", entry.Start, entry.End, entry.Target, p.PrintOperand(*entry.ErrorOp)))
		}
		p.decreaseIndent()
		p.writeLine("}")
	}
	p.decreaseIndent()
	p.writeIndent()
	p.write("}")
}

func (p *PrettyPrinter) printAnnotations(annotations values.AnnotationValues) {
	keys := make([]string, 0, len(annotations))
	for key := range annotations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		p.write("@")
		p.write(key)
		p.write("(")
		p.write(formatConstantValue(annotations[key]))
		p.write(") ")
	}
}

func (p *PrettyPrinter) PrintBasicBlock(basicBlock BIRBasicBlock) {
	p.writeLine(basicBlock.ID.Value() + " {")
	p.increaseIndent()
	for _, instruction := range basicBlock.Instructions {
		p.writeLine(p.PrintInstruction(instruction))
	}
	if basicBlock.Terminator != nil {
		p.writeLine(p.PrintInstruction(basicBlock.Terminator))
	}
	p.decreaseIndent()
	p.writeLine("}")
}

func (p *PrettyPrinter) PrintInstruction(instruction BIRInstruction) string {
	switch instruction := instruction.(type) {
	case *Move:
		return p.PrintMove(instruction)
	case *BinaryOp:
		return p.PrintBinaryOp(instruction)
	case *UnaryOp:
		return p.PrintUnaryOp(instruction)
	case *ConstantLoad:
		return p.PrintConstantLoad(instruction)
	case *Goto:
		return p.PrintGoto(instruction)
	case *Call:
		return p.PrintCall(instruction)
	case *Return:
		return p.PrintReturn(instruction)
	case *Branch:
		return p.PrintBranch(instruction)
	case *FieldAccess:
		return p.PrintFieldAccess(instruction)
	case *NewArray:
		return p.PrintNewArray(instruction)
	case *NewMap:
		return p.PrintNewMap(instruction)
	case *NewError:
		return p.PrintNewError(instruction)
	case *TypeCast:
		return p.PrintTypeCast(instruction)
	case *TypeTest:
		return p.PrintTypeTest(instruction)
	case *Panic:
		return p.PrintPanic(instruction)
	case *LockStart:
		return p.PrintLockStart(instruction)
	case *LockEnd:
		return p.PrintLockEnd(instruction)
	case *ResourceFunctionCall:
		return p.PrintResourceFunctionCall(instruction)
	case *StartAction:
		return p.PrintStartAction(instruction)
	case *SingleWaitAction:
		return p.PrintSingleWaitAction(instruction)
	case *AlternateWaitAction:
		return p.PrintAlternateWaitAction(instruction)
	case *MultipleWaitAction:
		return p.PrintMultipleWaitAction(instruction)
	case *NewObject:
		return p.PrintNewObject(instruction)
	case *NewStream:
		return p.PrintNewStream(instruction)
	case *StreamNext:
		return p.PrintStreamNext(instruction)
	case *StreamClose:
		return p.PrintStreamClose(instruction)
	case *FPLoad:
		return p.PrintFPLoad(instruction)
	case *PushScopeFrame:
		return p.PrintPushScopeFrame(instruction)
	case *PopScopeFrame:
		return "PopScopeFrame"
	case *NewXMLElement:
		return p.PrintNewXMLElement(instruction)
	case *NewXMLPI:
		return p.PrintNewXMLPI(instruction)
	case *NewXMLComment:
		return p.PrintNewXMLComment(instruction)
	case *NewXMLText:
		return p.PrintNewXMLText(instruction)
	case *NewXMLSequence:
		return p.PrintNewXMLSequence(instruction)
	case *EvalTemplateExpr:
		return p.PrintEvalTemplateExpr(instruction)
	case *XMLFilter:
		return p.PrintXMLFilter(instruction)
	default:
		panic(fmt.Sprintf("unknown instruction type: %T", instruction))
	}
}

func (p *PrettyPrinter) PrintFPLoad(fpLoad *FPLoad) string {
	kind := "fp"
	if fpLoad.IsClosure {
		kind = "closure_fp"
	}
	return fmt.Sprintf("%s = %s %s", p.PrintOperand(*fpLoad.LhsOp), kind, fpLoad.FunctionLookupKey)
}

func (p *PrettyPrinter) PrintPushScopeFrame(push *PushScopeFrame) string {
	return fmt.Sprintf("PushScopeFrame %d", push.NumLocals)
}

func (p *PrettyPrinter) PrintTypeCast(cast *TypeCast) string {
	return fmt.Sprintf("%s = <%s>(%s)", p.PrintOperand(*cast.LhsOp), semtypes.ToString(p.cx, cast.Type), p.PrintOperand(*cast.RhsOp))
}

func (p *PrettyPrinter) PrintTypeTest(test *TypeTest) string {
	op := "is"
	if test.IsNegation {
		op = "!is"
	}
	return fmt.Sprintf("%s = %s %s %s", p.PrintOperand(*test.LhsOp), p.PrintOperand(*test.RhsOp), op, semtypes.ToString(p.cx, test.Type))
}

func (p *PrettyPrinter) PrintNewArray(array *NewArray) string {
	values := strings.Builder{}
	for i, v := range array.Values {
		if i > 0 {
			values.WriteString(", ")
		}
		values.WriteString(p.PrintOperand(*v))
	}
	return fmt.Sprintf("%s = newArray %s[%s]{%s}", p.PrintOperand(*array.LhsOp), p.PrintSemType(array.Type), p.PrintOperand(*array.SizeOp), values.String())
}

func (p *PrettyPrinter) PrintNewMap(m *NewMap) string {
	values := strings.Builder{}
	for i, entry := range m.Values {
		if i > 0 {
			values.WriteString(", ")
		}
		if entry.IsKeyValuePair() {
			kv := entry.(*MappingConstructorKeyValueEntry)
			values.WriteString(p.PrintOperand(*kv.KeyOp()))
			values.WriteString("=")
			values.WriteString(p.PrintOperand(*kv.ValueOp()))
		} else {
			values.WriteString(p.PrintOperand(*entry.ValueOp()))
		}
	}
	defaults := strings.Builder{}
	for i, def := range m.Defaults {
		if i > 0 {
			defaults.WriteString(", ")
		}
		defaults.WriteString(def.FieldName)
		defaults.WriteString("=")
		defaults.WriteString(def.FunctionLookupKey)
	}
	if defaults.Len() > 0 {
		return fmt.Sprintf("%s = newMap %s{%s} defaults{%s}", p.PrintOperand(*m.LhsOp), p.PrintSemType(m.Type), values.String(), defaults.String())
	}
	return fmt.Sprintf("%s = newMap %s{%s}", p.PrintOperand(*m.LhsOp), p.PrintSemType(m.Type), values.String())
}

func (p *PrettyPrinter) PrintNewError(e *NewError) string {
	args := p.PrintOperand(*e.MessageOp)
	if e.CauseOp != nil {
		args += ", " + p.PrintOperand(*e.CauseOp)
	}
	if e.DetailOp != nil {
		args += ", " + p.PrintOperand(*e.DetailOp)
	}
	return fmt.Sprintf("%s = newError %s(%s)", p.PrintOperand(*e.LhsOp), p.PrintSemType(e.Type), args)
}

func (p *PrettyPrinter) PrintFieldAccess(access *FieldAccess) string {
	switch access.Kind {
	case InstructionKindMapStore, InstructionKindArrayStore, InstructionKindObjectStore:
		return fmt.Sprintf("%s[%s] = %s;", p.PrintOperand(*access.LhsOp), p.PrintOperand(*access.KeyOp), p.PrintOperand(*access.RhsOp))
	case InstructionKindMapLoad, InstructionKindArrayLoad, InstructionKindObjectLoad:
		return fmt.Sprintf("%s = %s[%s];", p.PrintOperand(*access.LhsOp), p.PrintOperand(*access.RhsOp), p.PrintOperand(*access.KeyOp))
	case InstructionKindArrayFillingLoad, InstructionKindMapFillingLoad:
		return fmt.Sprintf("%s = %s[%s] (fill);", p.PrintOperand(*access.LhsOp), p.PrintOperand(*access.RhsOp), p.PrintOperand(*access.KeyOp))
	default:
		panic(fmt.Sprintf("unknown field access kind: %d", access.Kind))
	}
}

func (p *PrettyPrinter) PrintNewObject(n *NewObject) string {
	return fmt.Sprintf("%s = newObject %s", p.PrintOperand(*n.LhsOp), n.ClassDefRef)
}

func (p *PrettyPrinter) PrintNewStream(n *NewStream) string {
	return fmt.Sprintf("%s = newStream %s %s", p.PrintOperand(*n.LhsOp), p.PrintSemType(n.StreamType), p.PrintOperand(*n.ImplOp))
}

func (p *PrettyPrinter) PrintStreamNext(n *StreamNext) string {
	return fmt.Sprintf("%s = streamNext %s", p.PrintOperand(*n.LhsOp), p.PrintOperand(*n.StreamOp))
}

func (p *PrettyPrinter) PrintStreamClose(n *StreamClose) string {
	return fmt.Sprintf("%s = streamClose %s", p.PrintOperand(*n.LhsOp), p.PrintOperand(*n.StreamOp))
}

func (p *PrettyPrinter) PrintClassDef(classDef BIRClassDef) {
	p.printAnnotations(classDef.Annotations)
	p.write("class ")
	p.write(classDef.Name.Value())
	p.write(" {\n")
	p.increaseIndent()
	for _, field := range classDef.Fields {
		p.writeLine(fmt.Sprintf("%s %s", field.Name, p.PrintSemType(field.Ty)))
	}
	var methodNames []string
	for name := range classDef.VTable {
		methodNames = append(methodNames, name)
	}
	sort.Strings(methodNames)
	for _, name := range methodNames {
		p.write("\n")
		p.writeIndent()
		p.PrintFunction(*classDef.VTable[name])
		p.write("\n")
	}
	var rmNames []string
	for name := range classDef.RTable {
		rmNames = append(rmNames, name)
	}
	sort.Strings(rmNames)
	for _, name := range rmNames {
		for _, entry := range classDef.RTable[name] {
			p.write("\n")
			p.writeIndent()
			p.write("resource ")
			p.write(name)
			p.write(" ")
			p.write(p.printResourcePath(entry))
			p.write(" ")
			p.PrintFunction(*entry.Fn)
			p.write("\n")
		}
	}
	p.decreaseIndent()
	p.write("}")
}

func literalPathSegment(seg ResourcePathSegmentDef) (string, bool) {
	shape := semtypes.SingleShape(seg.Ty)
	if !shape.IsPresent() {
		return "", false
	}
	s, ok := shape.Get().Value.(string)
	return s, ok
}

func (p *PrettyPrinter) printResourcePath(entry BIRResourceMethod) string {
	parts := []string{}
	for _, seg := range entry.PathSegments {
		if s, ok := literalPathSegment(seg); ok {
			parts = append(parts, s)
		} else {
			parts = append(parts, "["+p.PrintSemType(seg.Ty)+"]")
		}
	}
	if !semtypes.IsNever(entry.RestSegmentTy) {
		parts = append(parts, "["+p.PrintSemType(entry.RestSegmentTy)+"...]")
	}
	return strings.Join(parts, "/")
}

func (p *PrettyPrinter) PrintReturn(r *Return) string {
	return "return;"
}

func (p *PrettyPrinter) PrintPanic(pa *Panic) string {
	return fmt.Sprintf("panic %s;", p.PrintOperand(*pa.ErrorOp))
}

func (p *PrettyPrinter) PrintLockStart(l *LockStart) string {
	return fmt.Sprintf("lock-start %q GOTO %s;", l.LockKey, l.ThenBB.ID.Value())
}

func (p *PrettyPrinter) PrintLockEnd(l *LockEnd) string {
	return fmt.Sprintf("lock-end %q GOTO %s;", l.LockKey, l.ThenBB.ID.Value())
}

func (p *PrettyPrinter) PrintBranch(b *Branch) string {
	return fmt.Sprintf("%s ? %s : %s;", p.PrintOperand(*b.Op), b.TrueBB.ID.Value(), b.FalseBB.ID.Value())
}

func (p *PrettyPrinter) PrintGoto(g *Goto) string {
	return fmt.Sprintf("GOTO %s;", g.ThenBB.ID.Value())
}

func (p *PrettyPrinter) PrintResourceFunctionCall(call *ResourceFunctionCall) string {
	return fmt.Sprintf("%s = %s -> %s;", p.PrintOperand(*call.LhsOp), p.printCallSite(call.CallSite), call.ThenBB.ID.Value())
}

func (p *PrettyPrinter) PrintStartAction(action *StartAction) string {
	return fmt.Sprintf("%s = start %s isolated=%t -> %s;", p.PrintOperand(*action.LhsOp), p.printCallSite(action.Call), action.IsIsolated, action.ThenBB.ID.Value())
}

func (p *PrettyPrinter) PrintSingleWaitAction(action *SingleWaitAction) string {
	return fmt.Sprintf("%s = wait %s -> %s;", p.PrintOperand(*action.LhsOp), p.PrintOperand(action.Future), action.ThenBB.ID.Value())
}

func (p *PrettyPrinter) PrintAlternateWaitAction(action *AlternateWaitAction) string {
	futures := strings.Builder{}
	for i, future := range action.Futures {
		if i > 0 {
			futures.WriteString(" | ")
		}
		futures.WriteString(p.PrintOperand(future))
	}
	return fmt.Sprintf("%s = wait %s -> %s;", p.PrintOperand(*action.LhsOp), futures.String(), action.ThenBB.ID.Value())
}

func (p *PrettyPrinter) PrintMultipleWaitAction(action *MultipleWaitAction) string {
	fields := strings.Builder{}
	for i, future := range action.Futures {
		if i > 0 {
			fields.WriteString(", ")
		}
		fields.WriteString(action.FieldNames[i])
		fields.WriteString(": ")
		fields.WriteString(p.PrintOperand(future))
	}
	return fmt.Sprintf("%s = wait {%s} -> %s;", p.PrintOperand(*action.LhsOp), fields.String(), action.ThenBB.ID.Value())
}

func (p *PrettyPrinter) PrintCall(call *Call) string {
	return fmt.Sprintf("%s = %s -> %s;", p.PrintOperand(*call.LhsOp), p.printCallSite(call.CallSite), call.ThenBB.ID.Value())
}

func (p *PrettyPrinter) printCallSite(call CallSite) string {
	switch call.Kind {
	case CallKindFunction:
		return fmt.Sprintf("%s(%s)", call.Name.Value(), p.printOperands(call.Args))
	case CallKindFunctionPointer:
		name := call.Name.Value()
		if call.FpOperand != nil {
			name = p.PrintOperand(*call.FpOperand)
		}
		return fmt.Sprintf("%s(%s)", name, p.printOperands(call.Args))
	case CallKindMethod:
		return fmt.Sprintf("%s.%s(%s)", p.PrintOperand(*call.Receiver), call.Name.Value(), p.printOperands(call.Args[1:]))
	case CallKindResource:
		return fmt.Sprintf("%s->[%s].%s(%s)", p.PrintOperand(*call.Receiver), p.printOperands(call.PathSegments), call.MethodName, p.printOperands(call.Args))
	default:
		panic(fmt.Sprintf("unexpected call kind: %d", call.Kind))
	}
}

func (p *PrettyPrinter) printOperands(operands []BIROperand) string {
	result := strings.Builder{}
	for i, operand := range operands {
		if i > 0 {
			result.WriteString(",")
		}
		result.WriteString(p.PrintOperand(operand))
	}
	return result.String()
}

func (p *PrettyPrinter) PrintOperand(operand BIROperand) string {
	name := operand.VariableDcl.GetName()
	if operand.Address.Mode == AddressingModeAbsolute {
		return fmt.Sprintf("(%d, %s)", operand.Address.BaseIndex, name)
	}
	return name.Value()
}

func (p *PrettyPrinter) PrintConstantLoad(load *ConstantLoad) string {
	return fmt.Sprintf("%s = ConstantLoad %s", p.PrintOperand(*load.LhsOp), formatConstantValue(load.Value))
}

// formatConstantValue renders a BIR constant value for debug pretty printing.
// Primitive values keep Go's default `%v` format so the existing BIR text
// fixtures (which were captured with `%v`) remain stable. Composite Ballerina
// values such as list/tuple/map fillers are routed through values.String so
// they get a Ballerina-shaped form rather than a Go struct-pointer dump.
func formatConstantValue(v any) string {
	switch v.(type) {
	case *values.List, *values.Map, *values.Error, *values.Function, *values.Object, *values.TypeDesc:
		return values.String(v, map[uintptr]bool{})
	}
	if ref, ok := v.(*values.RuntimeAnnotationValueRef); ok {
		return fmt.Sprintf("runtime-ref(%s/%s:%s)", ref.Organization, ref.Module, ref.GlobalName)
	}
	return fmt.Sprintf("%v", v)
}

func (p *PrettyPrinter) PrintUnaryOp(op *UnaryOp) string {
	return fmt.Sprintf("%s = %s %s;", p.PrintOperand(*op.LhsOp), p.PrintInstructionKind(op.Kind), p.PrintOperand(*op.RhsOp))
}

func (p *PrettyPrinter) PrintBinaryOp(op *BinaryOp) string {
	return fmt.Sprintf("%s = %s %s %s;", p.PrintOperand(*op.LhsOp), p.PrintInstructionKind(op.Kind), p.PrintOperand(op.RhsOp1), p.PrintOperand(op.RhsOp2))
}

func (p *PrettyPrinter) PrintInstructionKind(kind InstructionKind) string {
	switch kind {
	case InstructionKindAdd:
		return "+"
	case InstructionKindSub:
		return "-"
	case InstructionKindMul:
		return "*"
	case InstructionKindDiv:
		return "/"
	case InstructionKindMod:
		return "%"
	case InstructionKindAnd:
		return "&&"
	case InstructionKindOr:
		return "||"
	case InstructionKindLessThan:
		return "<"
	case InstructionKindLessEqual:
		return "<="
	case InstructionKindGreaterThan:
		return ">"
	case InstructionKindGreaterEqual:
		return ">="
	case InstructionKindEqual:
		return "=="
	case InstructionKindNotEqual:
		return "!="
	case InstructionKindNot:
		return "!"
	case InstructionKindBitwiseComplement:
		return "~"
	case InstructionKindAnnotAccess:
		return ".@"
	}
	return "unknown"
}

func (p *PrettyPrinter) PrintMove(move *Move) string {
	return fmt.Sprintf("%s = %s;", p.PrintOperand(*move.LhsOp), p.PrintOperand(*move.RhsOp))
}

func (p *PrettyPrinter) PrintGlobalVar(globalVar BIRGlobalVariableDcl) string {
	sb := strings.Builder{}
	sb.WriteString(globalVar.Name.Value())
	sb.WriteString("  ")
	sb.WriteString(p.PrintSemType(globalVar.Type))
	return sb.String()
}

func (p *PrettyPrinter) PrintSemType(typeNode semtypes.SemType) string {
	if semtypes.IsZero(typeNode) {
		return "<UNKNOWN>"
	}
	return semtypes.ToString(p.cx, typeNode)
}

func (p *PrettyPrinter) PrintPackageID(packageID *model.PackageID) string {
	if packageID.IsUnnamed() {
		return "$anon-package"
	}
	orgName := string(*packageID.OrgName)
	pkgName := string(*packageID.PkgName)
	version := string(*packageID.Version)
	return fmt.Sprintf("%s.%s v %s", orgName, pkgName, version)
}

func (p *PrettyPrinter) PrintXMLFilter(n *XMLFilter) string {
	patterns := make([]string, len(n.Filters))
	for i, pattern := range n.Filters {
		switch pattern.Kind {
		case XMLNamePatternKindWildCard:
			patterns[i] = "*"
		case XMLNamePatternKindIdentifier:
			patterns[i] = fmt.Sprintf("{%q, %q}", "", pattern.Identifier)
		case XMLNamePatternKindQualifiedIdentifier:
			patterns[i] = fmt.Sprintf("{%q, %q}", pattern.NamespaceURI, pattern.Identifier)
		case XMLNamePatternKindPrefix:
			patterns[i] = fmt.Sprintf("{%q, *}", pattern.NamespaceURI)
		default:
			panic("unsupported XML name pattern kind")
		}
	}
	return fmt.Sprintf("%s = xmlFilter(%s, [%s])", p.PrintOperand(*n.LhsOp), p.PrintOperand(*n.Source), strings.Join(patterns, " | "))
}

func (p *PrettyPrinter) PrintNewXMLElement(n *NewXMLElement) string {
	children := "()"
	if n.ChildrenOp != nil {
		children = p.PrintOperand(*n.ChildrenOp)
	}
	attrs := "()"
	if n.AttrsOp != nil {
		attrs = p.PrintOperand(*n.AttrsOp)
	}
	qualifiedName := n.LocalName
	if n.Prefix != "" {
		qualifiedName = n.Prefix + ":" + n.LocalName
	}
	name := fmt.Sprintf("%q", qualifiedName)
	if n.NamespaceURI != "" {
		name = fmt.Sprintf("%s, namespace=%q", name, n.NamespaceURI)
	}
	if n.NamespacesOp != nil {
		return fmt.Sprintf("%s = newXMLElement(%s, %s, %s, %s)", p.PrintOperand(*n.LhsOp), name, children, attrs, p.PrintOperand(*n.NamespacesOp))
	}
	if n.AttrsOp != nil {
		return fmt.Sprintf("%s = newXMLElement(%s, %s, %s)", p.PrintOperand(*n.LhsOp), name, children, attrs)
	}
	return fmt.Sprintf("%s = newXMLElement(%s, %s)", p.PrintOperand(*n.LhsOp), name, children)
}

func (p *PrettyPrinter) PrintNewXMLPI(n *NewXMLPI) string {
	return fmt.Sprintf("%s = newXMLPI(%s, %s)", p.PrintOperand(*n.LhsOp), p.PrintOperand(*n.TargetOp), p.PrintOperand(*n.DataOp))
}

func (p *PrettyPrinter) PrintNewXMLComment(n *NewXMLComment) string {
	return fmt.Sprintf("%s = newXMLComment(%s)", p.PrintOperand(*n.LhsOp), p.PrintOperand(*n.BodyOp))
}

func (p *PrettyPrinter) PrintNewXMLText(n *NewXMLText) string {
	return fmt.Sprintf("%s = newXMLText(%s)", p.PrintOperand(*n.LhsOp), p.PrintOperand(*n.BodyOp))
}

func (p *PrettyPrinter) PrintEvalTemplateExpr(n *EvalTemplateExpr) string {
	kindStr := "string"
	if n.Kind == TemplateKindXML {
		kindStr = "xml"
	}
	parts := strings.Builder{}
	for i, s := range n.Strings {
		if i > 0 {
			parts.WriteString(", ")
		}
		fmt.Fprintf(&parts, "%q", s)
		if i < len(n.Insertions) {
			parts.WriteString(", ")
			parts.WriteString(p.PrintOperand(*n.Insertions[i]))
		}
	}
	return fmt.Sprintf("%s = evalTemplate[%s](%s)", p.PrintOperand(*n.LhsOp), kindStr, parts.String())
}

func (p *PrettyPrinter) PrintNewXMLSequence(n *NewXMLSequence) string {
	parts := strings.Builder{}
	for i, child := range n.Children {
		if i > 0 {
			parts.WriteString(", ")
		}
		parts.WriteString(p.PrintOperand(*child))
	}
	return fmt.Sprintf("%s = newXMLSequence{%s}", p.PrintOperand(*n.LhsOp), parts.String())
}
