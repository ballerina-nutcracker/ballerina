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

package exec

import (
	"fmt"
	"math"
	"unsafe"

	"ballerina-lang-go/bir"
	"ballerina-lang-go/decimal"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/runtime/internal/modules"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

func execConstantLoad(ctx *extern.Context, constantLoad *bir.ConstantLoad, frame *Frame) {
	setOperandValue(ctx, constantLoad.LhsOp, frame, constantLoad.Value)
}

func execMove(ctx *extern.Context, moveIns *bir.Move, frame *Frame) {
	setOperandValue(ctx, moveIns.LhsOp, frame, getOperandValue(ctx, moveIns.RhsOp, frame))
}

func execNewArray(ctx *extern.Context, newArray *bir.NewArray, frame *Frame) {
	size := 0
	if newArray.SizeOp != nil {
		size = int(getOperandValue(ctx, newArray.SizeOp, frame).(int64))
	}
	initial := make([]values.BalValue, len(newArray.Values))
	for i, value := range newArray.Values {
		initial[i] = getOperandValue(ctx, value, frame)
	}
	atomic := semtypes.ToListAtomicType(ctx.TypeCtx, newArray.Type)
	list := values.NewList(newArray.Type, atomic, newArray.IsReadonly, newArray.Filler, size, initial)
	setOperandValue(ctx, newArray.LhsOp, frame, list)
}

func execNewMap(ctx *extern.Context, newMap *bir.NewMap, frame *Frame) {
	seen := make(map[string]struct{}, len(newMap.Values))
	entries := make([]values.MapEntry, 0, len(newMap.Values)+len(newMap.Defaults))
	for _, entry := range newMap.Values {
		kv := entry.(*bir.MappingConstructorKeyValueEntry)
		keyStr := getOperandValue(ctx, kv.KeyOp(), frame).(string)
		valueVal := getOperandValue(ctx, kv.ValueOp(), frame)
		seen[keyStr] = struct{}{}
		entries = append(entries, values.MapEntry{Key: keyStr, Value: valueVal})
	}
	for _, def := range newMap.Defaults {
		if _, exists := seen[def.FieldName]; exists {
			continue
		}
		fn := ctx.Env.Registry.(*modules.Registry).GetBIRFunction(def.FunctionLookupKey)
		val := executeFunction(ctx, fn, nil, frame)
		entries = append(entries, values.MapEntry{Key: def.FieldName, Value: val})
	}
	atomic := semtypes.ToMappingAtomicType(ctx.TypeCtx, newMap.Type)
	if atomic == nil {
		panic("mapping inherent type has no atomic representation")
	}
	m := values.NewMap(newMap.Type, atomic, newMap.IsReadonly, entries)
	setOperandValue(ctx, newMap.GetLhsOperand(), frame, m)
}

func execNewError(ctx *extern.Context, newError *bir.NewError, frame *Frame) {
	msgVal := getOperandValue(ctx, newError.MessageOp, frame)
	message := msgVal.(string)

	var cause values.BalValue
	if newError.CauseOp != nil {
		cause = getOperandValue(ctx, newError.CauseOp, frame)
	}

	var detailMap *values.Map
	if newError.DetailOp != nil {
		detailMap = getOperandValue(ctx, newError.DetailOp, frame).(*values.Map)
	}
	errVal := values.NewError(newError.Type, message, cause, newError.TypeName, detailMap)
	setOperandValue(ctx, newError.GetLhsOperand(), frame, errVal)
}

func execNewObject(ctx *extern.Context, newObject *bir.NewObject, frame *Frame) {
	classDef := ctx.Env.Registry.(*modules.Registry).GetClassDef(newObject.ClassDefRef)
	fieldValues := make(map[string]values.BalValue, len(classDef.Fields))
	methodKeys := make(map[string]string, len(classDef.VTable))
	for methodName, method := range classDef.VTable {
		methodKeys[methodName] = method.FunctionLookupKey
	}
	rtable := make(map[string][]values.ResourceEntry, len(classDef.RTable))
	for methodName, entries := range classDef.RTable {
		copied := make([]values.ResourceEntry, len(entries))
		for i, entry := range entries {
			segs := make([]values.ResourcePathSegmentDef, len(entry.PathSegments))
			for j, seg := range entry.PathSegments {
				segs[j] = values.ResourcePathSegmentDef{Ty: seg.Ty}
			}
			copied[i] = values.ResourceEntry{
				PathSegments:      segs,
				RestSegmentTy:     entry.RestSegmentTy,
				FunctionLookupKey: entry.Fn.FunctionLookupKey,
			}
		}
		rtable[methodName] = copied
	}
	objType := newObject.GetLhsOperand().VariableDcl.GetType()
	obj := values.NewObject(objType, fieldValues, methodKeys, rtable)
	setOperandValue(ctx, newObject.GetLhsOperand(), frame, obj)
}

func execArrayStore(ctx *extern.Context, access *bir.FieldAccess, frame *Frame) {
	list := getOperandValue(ctx, access.LhsOp, frame).(*values.List)
	idx := int(getOperandValue(ctx, access.KeyOp, frame).(int64))
	if idx < 0 {
		panic(values.NewErrorWithMessage(fmt.Sprintf("invalid array index: %d", idx)))
	}
	list.FillingSet(ctx.TypeCtx, idx, getOperandValue(ctx, access.RhsOp, frame))
}

func execArrayLoad(ctx *extern.Context, access *bir.FieldAccess, frame *Frame) {
	list := getOperandValue(ctx, access.RhsOp, frame).(*values.List)
	idx := int(getOperandValue(ctx, access.KeyOp, frame).(int64))
	if idx < 0 || idx >= list.Len() {
		panic(values.NewErrorWithMessage(fmt.Sprintf("invalid array index: %d", idx)))
	}
	setOperandValue(ctx, access.LhsOp, frame, list.Get(idx))
}

func execArrayFillingLoad(ctx *extern.Context, access *bir.FieldAccess, frame *Frame) {
	list := getOperandValue(ctx, access.RhsOp, frame).(*values.List)
	idx := int(getOperandValue(ctx, access.KeyOp, frame).(int64))
	if idx < 0 {
		panic(values.NewErrorWithMessage(fmt.Sprintf("invalid array index: %d", idx)))
	}
	setOperandValue(ctx, access.LhsOp, frame, list.FillingGet(idx))
}

func execMapStore(ctx *extern.Context, access *bir.FieldAccess, frame *Frame) {
	container := getOperandValue(ctx, access.LhsOp, frame)
	keyStr := getOperandValue(ctx, access.KeyOp, frame).(string)
	if container == nil {
		panic(values.NewErrorWithMessage(fmt.Sprintf("missing key: %q", keyStr)))
	}
	m := container.(*values.Map)
	valueVal := getOperandValue(ctx, access.RhsOp, frame)
	if valueVal == nil && m.ShouldDeleteOnNilStore(ctx.TypeCtx, keyStr) {
		m.Delete(ctx.TypeCtx, keyStr)
		return
	}
	m.Put(ctx.TypeCtx, keyStr, valueVal)
}

func execMapFillingLoad(ctx *extern.Context, access *bir.FieldAccess, frame *Frame) {
	container := getOperandValue(ctx, access.RhsOp, frame)
	key := getOperandValue(ctx, access.KeyOp, frame).(string)
	if container == nil {
		panic(values.NewErrorWithMessage(fmt.Sprintf("missing key: %q", key)))
	}
	setOperandValue(ctx, access.LhsOp, frame, container.(*values.Map).FillingGet(ctx.TypeCtx, key, access.Filler))
}

func execMapLoad(ctx *extern.Context, access *bir.FieldAccess, frame *Frame) {
	container := getOperandValue(ctx, access.RhsOp, frame)
	key := getOperandValue(ctx, access.KeyOp, frame).(string)
	if container == nil {
		setOperandValue(ctx, access.LhsOp, frame, nil)
		return
	}
	value, _ := container.(*values.Map).Get(key)
	setOperandValue(ctx, access.LhsOp, frame, value)
}

func execObjectStore(ctx *extern.Context, access *bir.FieldAccess, frame *Frame) {
	obj := getOperandValue(ctx, access.LhsOp, frame).(*values.Object)
	field := getOperandValue(ctx, access.KeyOp, frame).(string)
	value := getOperandValue(ctx, access.RhsOp, frame)
	obj.Put(field, value)
}

func execObjectLoad(ctx *extern.Context, access *bir.FieldAccess, frame *Frame) {
	obj := getOperandValue(ctx, access.RhsOp, frame).(*values.Object)
	field := getOperandValue(ctx, access.KeyOp, frame).(string)
	value, _ := obj.Get(field)
	setOperandValue(ctx, access.LhsOp, frame, value)
}

func execTypeCast(ctx *extern.Context, typeCast *bir.TypeCast, frame *Frame) {
	sourceValue := getOperandValue(ctx, typeCast.RhsOp, frame)
	result := castValue(ctx, sourceValue, typeCast.Type)
	setOperandValue(ctx, typeCast.LhsOp, frame, result)
}

func execFPLoad(ctx *extern.Context, fpLoad *bir.FPLoad, frame *Frame) {
	fn := &values.Function{
		Type:      fpLoad.Type,
		LookupKey: fpLoad.FunctionLookupKey,
	}
	if fpLoad.IsClosure {
		frame.MarkEscaped()
		fn.ParentFrame = frame
	}
	setOperandValue(ctx, fpLoad.LhsOp, frame, fn)
}

func execTypeTest(ctx *extern.Context, typeTest *bir.TypeTest, frame *Frame) {
	sourceValue := getOperandValue(ctx, typeTest.RhsOp, frame)
	valueType := values.SemTypeForValue(sourceValue)
	typeCtx := ctx.TypeCtx
	matches := semtypes.IsSubtype(typeCtx, valueType, typeTest.Type) != typeTest.IsNegation
	setOperandValue(ctx, typeTest.LhsOp, frame, matches)
}

func castValue(ctx *extern.Context, value values.BalValue, targetType semtypes.SemType) values.BalValue {
	typeCtx := ctx.TypeCtx
	valueType := values.SemTypeForValue(value)
	if semtypes.IsSubtype(typeCtx, valueType, targetType) {
		return value
	}
	var converted values.BalValue
	switch {
	case semtypes.IsSubtypeSimple(targetType, semtypes.INT):
		converted = toInt(value)
	case semtypes.IsSubtypeSimple(targetType, semtypes.FLOAT):
		converted = toFloat(value)
	case semtypes.IsSubtypeSimple(targetType, semtypes.DECIMAL):
		converted = toDecimal(value)
	default:
		panic(badTypeCastError())
	}
	// Numeric conversion only guarantees the basic type; narrow subtypes
	// (e.g. `2|3|4`, `int:Signed8`, `byte`) still require a membership check.
	if !semtypes.IsSubtype(typeCtx, values.SemTypeForValue(converted), targetType) {
		panic(badTypeCastError())
	}
	return converted
}

func badTypeCastError() *values.Error {
	return values.NewErrorWithMessage("bad type cast")
}

func toInt(value any) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			panic(values.NewErrorWithMessage(fmt.Sprintf("bad type cast: cannot cast non-finite value %v to int", v)))
		}
		if v < float64(math.MinInt64) || v > float64(math.MaxInt64) {
			panic(values.NewErrorWithMessage(fmt.Sprintf("bad type cast: cannot cast out-of-range value %v to int", v)))
		}
		return int64(math.RoundToEven(v))
	case *decimal.Decimal:
		return decimalToInt(v)
	default:
		panic(values.NewErrorWithMessage(fmt.Sprintf("bad type cast: cannot cast %v to int", value)))
	}
}

func decimalToInt(v *decimal.Decimal) int64 {
	n, ok, err := v.Int64()
	if err != nil {
		panic(values.NewErrorWithMessage(fmt.Sprintf("cannot convert %v to int: %v", v, err)))
	}
	if !ok {
		panic(values.NewErrorWithMessage(fmt.Sprintf("cannot convert %v to int64: value out of range", v)))
	}
	return n
}

func toFloat(value any) float64 {
	switch v := value.(type) {
	case int64:
		return float64(v)
	case float64:
		return v
	case *decimal.Decimal:
		return v.Float64()
	default:
		panic(values.NewErrorWithMessage(fmt.Sprintf("bad type cast: cannot cast %v to float", value)))
	}
}

// floatToDecimal converts an IEEE 754 float64 into a Ballerina decimal.
// Ballerina decimals do not support NaN, infinities, or subnormals, so any
// such input triggers a runtime panic with the spec-mandated message.
func floatToDecimal(v float64) *decimal.Decimal {
	d, err := decimal.FromFloat64(v)
	if err != nil {
		panic(values.NewErrorWithMessage(err.Error()))
	}
	return d
}

func toDecimal(value any) *decimal.Decimal {
	switch v := value.(type) {
	case int64:
		return decimal.FromInt64(v)
	case float64:
		return floatToDecimal(v)
	case *decimal.Decimal:
		return v
	default:
		panic(values.NewErrorWithMessage(fmt.Sprintf("bad type cast: cannot cast %v to decimal", value)))
	}
}

func execNewXMLText(ctx *extern.Context, instr *bir.NewXMLText, frame *Frame) {
	body := getOperandValue(ctx, instr.BodyOp, frame).(string)
	setOperandValue(ctx, instr.LhsOp, frame, values.NewXMLText(body))
}

func execNewXMLComment(ctx *extern.Context, instr *bir.NewXMLComment, frame *Frame) {
	body := getOperandValue(ctx, instr.BodyOp, frame).(string)
	setOperandValue(ctx, instr.LhsOp, frame, values.NewXMLComment(body, xmlResultReadonly(ctx, instr.LhsOp)))
}

func execNewXMLPI(ctx *extern.Context, instr *bir.NewXMLPI, frame *Frame) {
	target := getOperandValue(ctx, instr.TargetOp, frame).(string)
	data := getOperandValue(ctx, instr.DataOp, frame).(string)
	setOperandValue(ctx, instr.LhsOp, frame, values.NewXMLProcessingInstruction(target, data, xmlResultReadonly(ctx, instr.LhsOp)))
}

func execNewXMLElement(ctx *extern.Context, instr *bir.NewXMLElement, frame *Frame) {
	name := getOperandValue(ctx, instr.NameOp, frame).(string)
	var children values.XMLValue
	if instr.ChildrenOp != nil {
		raw := getOperandValue(ctx, instr.ChildrenOp, frame)
		v, ok := raw.(values.XMLValue)
		if !ok {
			panic(fmt.Sprintf("invariant violation: NewXMLElement children operand %v is not an XMLValue (got %T)", instr.ChildrenOp, raw))
		}
		children = v
	}
	var attrs *values.Map
	if instr.AttrsOp != nil {
		attrs = getOperandValue(ctx, instr.AttrsOp, frame).(*values.Map)
	}
	var namespaces *values.Map
	if instr.NamespacesOp != nil {
		namespaces = getOperandValue(ctx, instr.NamespacesOp, frame).(*values.Map)
	}
	setOperandValue(ctx, instr.LhsOp, frame, values.NewXMLElement(name, attrs, namespaces, children, xmlResultReadonly(ctx, instr.LhsOp)))
}

func xmlResultReadonly(ctx *extern.Context, op *bir.BIROperand) bool {
	return semtypes.IsSubtype(ctx.TypeCtx, op.VariableDcl.GetType(), semtypes.VAL_READONLY)
}

func execEvalTemplateExpr(ctx *extern.Context, instr *bir.EvalTemplateExpr, frame *Frame) {
	n := len(instr.Insertions)
	buf := make([]byte, 0, instr.LiteralsTotalLen+8*n)
	for i := 0; i < n; i++ {
		buf = append(buf, instr.Strings[i]...)
		buf = append(buf, values.String(getOperandValue(ctx, instr.Insertions[i], frame), nil)...)
	}
	buf = append(buf, instr.Strings[n]...)
	var result string
	if len(buf) > 0 {
		result = unsafe.String(&buf[0], len(buf))
	}
	switch instr.Kind {
	case bir.TemplateKindString:
		setOperandValue(ctx, instr.LhsOp, frame, result)
	case bir.TemplateKindXML:
		xmlValue, err := values.ParseAsXMLValue(ctx.TypeCtx, result, values.XMLTemplateMode)
		if err != nil {
			panic(err)
		}
		setOperandValue(ctx, instr.LhsOp, frame, xmlValue)
	default:
		panic(fmt.Sprintf("unsupported template kind: %d", instr.Kind))
	}
}

func execNewXMLSequence(ctx *extern.Context, instr *bir.NewXMLSequence, frame *Frame) {
	items := make([]values.XMLValue, 0, len(instr.Children))
	for i, op := range instr.Children {
		val := getOperandValue(ctx, op, frame)
		x, ok := val.(values.XMLValue)
		if !ok {
			panic(fmt.Sprintf("invariant violation: NewXMLSequence child %d operand %v is not an XMLValue (got %T)", i, op, val))
		}
		items = append(items, x)
	}
	setOperandValue(ctx, instr.LhsOp, frame, values.NewNormalizedXMLSequence(items))
}
