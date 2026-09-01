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

package xml

import (
	"fmt"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName        = "ballerina"
	moduleName     = "lang.xml"
	nextMethodName = "$xmlIterator.next"
)

type xmlExtern = extern.NativeFunc

func initXMLModule(rt *runtime.Runtime) {
	functions := map[string]xmlExtern{
		"length": length, "iterator": xmlIterator, "get": get, "concat": concat,
		"getName": getName, "setName": setName, "getAttributes": getAttributes,
		"getChildren": getChildren, "setChildren": setChildren, "getDescendants": getDescendants,
		"data": data, "getTarget": getTarget, "getContent": getContent,
		"createElement": createElement, "createProcessingInstruction": createProcessingInstruction,
		"createComment": createComment, "createText": createText, "slice": slice,
		"strip": strip, "elements": elements, "children": children,
		"elementChildren": elementChildren, "text": text, "map": xmlMap,
		"forEach": forEach, "filter": filter, "fromString": fromString,
		nextMethodName: xmlIteratorNext,
	}
	for name, function := range functions {
		runtime.RegisterExternFunction(rt, orgName, moduleName, name, function)
	}
}

func asXML(value values.BalValue) values.XMLValue { return value.(values.XMLValue) }

func length(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return int64(len(asXML(args[0]).IterItems())), nil
}

func xmlIterator(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	xmlValue := asXML(args[0])
	iteratorTy := xmlIteratorType(ctx.TypeEnv(), semtypes.XMLItemType(xmlValue.Type()))
	return values.NewObject(iteratorTy, map[string]values.BalValue{"items": xmlValue.IterItems(), "idx": int64(0)}, map[string]string{"next": orgName + "/" + moduleName + ":" + nextMethodName}, nil, nil), nil
}

func xmlIteratorNext(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	it := args[0].(*values.Object)
	itemsValue, _ := it.Get("items")
	idxValue, _ := it.Get("idx")
	items, idx := itemsValue.([]values.XMLValue), idxValue.(int64)
	if idx >= int64(len(items)) {
		return nil, nil
	}
	it.Put("idx", idx+1)
	recordTy := xmlIteratorNextRecordType(ctx.TypeCtx(), it.Type)
	return values.NewMap(recordTy, semtypes.ToMappingAtomicType(ctx.TypeCtx(), recordTy), false, []values.MapEntry{{Key: "value", Value: items[idx]}}), nil
}

func xmlIteratorType(env semtypes.Env, itemTy semtypes.SemType) semtypes.SemType {
	recordDef := semtypes.NewMappingDefinition()
	recordTy := recordDef.Define(env, []semtypes.Field{semtypes.FieldFrom("value", itemTy, false, false)}, semtypes.Never)
	emptyParamsDef := semtypes.NewListDefinition()
	emptyParamsTy := emptyParamsDef.Define(env, nil, semtypes.ListMutability(semtypes.CellMutabilityNone))
	nextDef := semtypes.NewFunctionDefinition()
	nextTy := nextDef.Define(env, emptyParamsTy, semtypes.Union(recordTy, semtypes.Nil), semtypes.FunctionQualifiersFrom(env, true, false))
	iteratorDef := semtypes.NewObjectDefinition()
	return iteratorDef.Define(env, semtypes.ObjectQualifiersDefault, []semtypes.Member{{
		Name: "next", ValueType: nextTy, Kind: semtypes.MemberKindMethod, Visibility: semtypes.VisibilityPublic, Immutable: true,
	}})
}

func xmlIteratorNextRecordType(ctx semtypes.Context, iteratorTy semtypes.SemType) semtypes.SemType {
	nextTy := semtypes.ObjectMemberType(ctx, semtypes.StringConst("next"), iteratorTy)
	emptyArgsDef := semtypes.NewListDefinition()
	emptyArgsTy := emptyArgsDef.Define(ctx.Env(), nil, semtypes.ListMutability(semtypes.CellMutabilityNone))
	return semtypes.Diff(semtypes.FunctionReturnType(ctx, nextTy, emptyArgsTy), semtypes.Nil)
}

func get(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	items, index := asXML(args[0]).IterItems(), args[1].(int64)
	if index < 0 || index >= int64(len(items)) {
		panic(values.NewErrorWithMessage("XML index out of range"))
	}
	return items[index], nil
}

func concat(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	items := make([]values.XMLValue, 0, len(args))
	for _, arg := range args {
		switch value := arg.(type) {
		case string:
			if err := values.ValidateXMLCharacters(value); err != nil {
				panic(values.NewErrorWithMessage(err.Error()))
			}
			items = append(items, values.NewXMLText(value))
		case values.XMLValue:
			items = append(items, value)
		}
	}
	return values.NewNormalizedXMLSequence(items), nil
}

func getName(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return args[0].(*values.XMLElement).ExpandedName(), nil
}

func setName(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	args[0].(*values.XMLElement).SetExpandedName(args[1].(string))
	return nil, nil
}

func getAttributes(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	element := args[0].(*values.XMLElement)
	entries := make([]values.MapEntry, 0, element.Attributes.Len()+element.Namespaces.Len())
	for _, key := range element.Attributes.Keys() {
		value, _ := element.Attributes.Get(key)
		entries = append(entries, values.MapEntry{Key: key, Value: value})
	}
	for _, key := range element.Namespaces.Keys() {
		value, _ := element.Namespaces.Get(key)
		local := "xmlns"
		if strings.HasPrefix(key, "xmlns:") {
			local = strings.TrimPrefix(key, "xmlns:")
		}
		entries = append(entries, values.MapEntry{Key: values.ExpandedXMLName(values.XMLNSNamespaceURI, local), Value: value})
	}
	return values.NewXMLStringMap(ctx.TypeCtx(), false, entries), nil
}

func getChildren(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return args[0].(*values.XMLElement).Children, nil
}

func setChildren(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	var children values.XMLValue
	switch value := args[1].(type) {
	case string:
		if err := values.ValidateXMLCharacters(value); err != nil {
			panic(values.NewErrorWithMessage(err.Error()))
		}
		children = values.NewXMLText(value)
	case values.XMLValue:
		children = value
	}
	args[0].(*values.XMLElement).SetXMLChildren(children)
	return nil, nil
}

func appendDescendants(out *[]values.XMLValue, value values.XMLValue) {
	for _, item := range value.IterItems() {
		*out = append(*out, item)
		if element, ok := item.(*values.XMLElement); ok {
			appendDescendants(out, element.Children)
		}
	}
}

func getDescendants(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	var result []values.XMLValue
	appendDescendants(&result, args[0].(*values.XMLElement).Children)
	return values.NewNormalizedXMLSequence(result), nil
}

func appendData(b *strings.Builder, value values.XMLValue) {
	for _, item := range value.IterItems() {
		switch item := item.(type) {
		case *values.XMLText:
			b.WriteString(item.Body)
		case *values.XMLElement:
			appendData(b, item.Children)
		}
	}
}

func data(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	var result strings.Builder
	appendData(&result, asXML(args[0]))
	return result.String(), nil
}

func getTarget(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return args[0].(*values.XMLProcessingInstruction).Target, nil
}

func getContent(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	switch value := args[0].(type) {
	case *values.XMLProcessingInstruction:
		return value.Data, nil
	case *values.XMLComment:
		return value.Body, nil
	}
	panic("unreachable")
}

func createElement(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	name, source, child := args[0].(string), args[1].(*values.Map), asXML(args[2])
	uri, local, err := values.ParseExpandedXMLName(name)
	if err != nil || uri == values.XMLNSNamespaceURI {
		panic(values.NewErrorWithMessage("invalid XML element name"))
	}
	var attrs, namespaces []values.MapEntry
	for _, key := range source.Keys() {
		raw, _ := source.Get(key)
		value := raw.(string)
		if err := values.ValidateXMLCharacters(value); err != nil {
			panic(values.NewErrorWithMessage(err.Error()))
		}
		aURI, aLocal, err := values.ParseExpandedXMLName(key)
		if err != nil {
			panic(values.NewErrorWithMessage(err.Error()))
		}
		if aURI == values.XMLNSNamespaceURI {
			prefix, internalKey := aLocal, "xmlns:"+aLocal
			if aLocal == "xmlns" {
				prefix, internalKey = "", "xmlns"
			}
			if err := validateNamespace(prefix, value); err != nil {
				panic(values.NewErrorWithMessage(err.Error()))
			}
			if prefix == "" && value != "" && uri == "" {
				panic(values.NewErrorWithMessage("unnamespaced XML element conflicts with its local default namespace"))
			}
			namespaces = append(namespaces, values.MapEntry{Key: internalKey, Value: value})
			continue
		}
		if aURI == "" && aLocal == "xmlns" {
			panic(values.NewErrorWithMessage("xmlns must be a namespace attribute"))
		}
		attrs = append(attrs, values.MapEntry{Key: key, Value: value})
	}
	return values.NewXMLElement(ctx.TypeCtx(), "", local, uri, values.NewXMLStringMap(ctx.TypeCtx(), false, attrs), values.NewXMLStringMap(ctx.TypeCtx(), false, namespaces), values.NewNormalizedXMLSequence([]values.XMLValue{child}), false), nil
}

func validateNamespace(prefix, uri string) error {
	if prefix == "xmlns" || uri == values.XMLNSNamespaceURI || prefix != "" && uri == "" || prefix == "xml" && uri != values.XMLNamespaceURI || prefix != "xml" && uri == values.XMLNamespaceURI {
		return fmt.Errorf("invalid XML namespace declaration")
	}
	return nil
}

func createProcessingInstruction(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return values.NewValidatedXMLProcessingInstruction(args[0].(string), args[1].(string), false), nil
}

func createComment(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return values.NewValidatedXMLComment(args[0].(string), false), nil
}

func createText(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	if err := values.ValidateXMLCharacters(args[0].(string)); err != nil {
		panic(values.NewErrorWithMessage(err.Error()))
	}
	return values.NewNormalizedXMLSequence([]values.XMLValue{values.NewXMLText(args[0].(string))}), nil
}

func slice(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	items := asXML(args[0]).IterItems()
	start, end := args[1].(int64), args[2].(int64)
	if start < 0 || end < start || end > int64(len(items)) {
		panic(values.NewErrorWithMessage("invalid XML slice bounds"))
	}
	return values.NewNormalizedXMLSequence(items[start:end]), nil
}

func strip(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	var result []values.XMLValue
	for _, item := range asXML(args[0]).IterItems() {
		switch item := item.(type) {
		case *values.XMLComment, *values.XMLProcessingInstruction:
		case *values.XMLText:
			if strings.Trim(item.Body, " \t\r\n") != "" {
				result = append(result, item)
			}
		default:
			result = append(result, item)
		}
	}
	return values.NewNormalizedXMLSequence(result), nil
}

func selectedElements(value values.XMLValue, name values.BalValue) values.XMLValue {
	var uri, local string
	if name != nil {
		var err error
		uri, local, err = values.ParseExpandedXMLName(name.(string))
		if err != nil || uri == values.XMLNSNamespaceURI {
			panic(values.NewErrorWithMessage("invalid XML element name filter"))
		}
	}
	var result []values.XMLValue
	for _, item := range value.IterItems() {
		if element, ok := item.(*values.XMLElement); ok && (name == nil || element.NamespaceURI == uri && element.LocalName == local) {
			result = append(result, element)
		}
	}
	return values.NewNormalizedXMLSequence(result)
}

func elements(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return selectedElements(asXML(args[0]), args[1]), nil
}

func directChildren(value values.XMLValue) values.XMLValue {
	var result []values.XMLValue
	for _, item := range value.IterItems() {
		if element, ok := item.(*values.XMLElement); ok {
			result = append(result, element.Children)
		}
	}
	return values.NewNormalizedXMLSequence(result)
}

func children(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return directChildren(asXML(args[0])), nil
}

func elementChildren(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return selectedElements(directChildren(asXML(args[0])), args[1]), nil
}

func text(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	var result strings.Builder
	for _, item := range asXML(args[0]).IterItems() {
		if text, ok := item.(*values.XMLText); ok {
			result.WriteString(text.Body)
		}
	}
	return values.NewNormalizedXMLSequence([]values.XMLValue{values.NewXMLText(result.String())}), nil
}

func xmlMap(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	callback := args[1].(*values.Function)
	var result []values.XMLValue
	for _, item := range asXML(args[0]).IterItems() {
		mapped, err := ctx.InvokeFunctionValue(callback, []values.BalValue{item})
		if err != nil {
			return nil, err
		}
		result = append(result, asXML(mapped))
	}
	return values.NewNormalizedXMLSequence(result), nil
}

func forEach(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	callback := args[1].(*values.Function)
	for _, item := range asXML(args[0]).IterItems() {
		if _, err := ctx.InvokeFunctionValue(callback, []values.BalValue{item}); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func filter(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	callback := args[1].(*values.Function)
	var result []values.XMLValue
	for _, item := range asXML(args[0]).IterItems() {
		keep, err := ctx.InvokeFunctionValue(callback, []values.BalValue{item})
		if err != nil {
			return nil, err
		}
		if keep.(bool) {
			result = append(result, item)
		}
	}
	return values.NewNormalizedXMLSequence(result), nil
}

func fromString(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	result, err := values.ParseAsXMLValue(ctx.TypeCtx(), args[0].(string), values.XMLStrictContentMode)
	if err != nil {
		return values.NewErrorWithMessage("lang.xml:fromString: " + err.Error()), nil
	}
	return result, nil
}

func init() { runtime.RegisterModuleInitializer(initXMLModule) }
