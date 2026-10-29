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

package native

import (
	"errors"
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "data.xmldata"
)

func init() {
	runtime.RegisterModuleInitializer(initXmldataModule)
}

func initXmldataModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "parseAsType", parseAsType)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "parseString", parseString)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "parseBytes", parseBytes)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "parseStream", parseStream)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "validate", validate)
}

func parseAsType(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	source, _ := args[0].(values.XMLValue)
	opts, _ := args[1].(*values.Map)
	td, _ := args[2].(*values.TypeDesc)
	t := newTyperFor(ctx, td)
	result, err := traverseXML(t, readSourceOptions(opts), source, t.root)
	return toResult(ctx, result, err)
}

func parseString(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	input, _ := args[0].(string)
	return parseText(ctx, input, args[1], args[2])
}

func parseBytes(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	input, _ := args[0].(*values.List)
	return parseText(ctx, string(input.ToByteSlice()), args[1], args[2])
}

// parseStream reads the whole byte-block stream before parsing.
func parseStream(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	byteStream, _ := args[0].(*values.Stream)
	var input []byte
	for {
		next := byteStream.Next()
		if streamErr, ok := next.(*values.Error); ok {
			if byteStream.Close != nil {
				byteStream.Close()
			}
			return toResult(ctx, nil, errors.New(streamErr.Message))
		}
		record, ok := next.(*values.Map)
		if !ok {
			break
		}
		block, _ := record.Get("value")
		input = append(input, block.(*values.List).ToByteSlice()...)
	}
	return parseText(ctx, string(input), args[1], args[2])
}

func parseText(ctx *extern.Context, input string, optsArg, tdArg values.BalValue) (values.BalValue, error) {
	opts, _ := optsArg.(*values.Map)
	td, _ := tdArg.(*values.TypeDesc)
	t := newTyperFor(ctx, td)
	result, err := parseXML(t, readSourceOptions(opts), input, t.root)
	return toResult(ctx, result, err)
}

// validate checks the XML against a record type by converting it with the default options. Validating
// against an XSD file is not supported.
func validate(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	source, _ := args[0].(values.XMLValue)
	td, ok := args[1].(*values.TypeDesc)
	if !ok {
		return toResult(ctx, nil, invalidXMLError("validating XML against an XSD file is not supported"))
	}
	t := newTyperFor(ctx, td)
	opts := sourceOptions{textFieldName: contentField, allowDataProjection: true, useSemanticEquality: true}
	if _, err := traverseXML(t, opts, source, t.root); err != nil {
		return toResult(ctx, nil, invalidXMLError(err.Error()))
	}
	return nil, nil
}

// newTyperFor drops error from the typedesc constraint: an inferred typedesc can wrongly carry the module's
// distinct error type (ballerina-nutcracker/ballerina#1383).
func newTyperFor(ctx *extern.Context, td *values.TypeDesc) *typer {
	annotations, _ := ctx.TypeAnnotations(td)
	annots := typeAnnotations{typeLevel: annotations.Annotations, fields: annotations.Fields}
	return newTyper(ctx.TypeCtx(), semtypes.Diff(td.Constraint(), semtypes.Error), annots)
}

func readSourceOptions(opts *values.Map) sourceOptions {
	attributePrefix, _ := opts.Get("attributePrefix")
	textFieldName, _ := opts.Get("textFieldName")
	allowDataProjection, _ := opts.Get("allowDataProjection")
	useSemanticEquality, _ := opts.Get("useSemanticEquality")
	return sourceOptions{
		attributePrefix:     fmt.Sprint(attributePrefix),
		textFieldName:       fmt.Sprint(textFieldName),
		allowDataProjection: allowDataProjection == true,
		useSemanticEquality: useSemanticEquality == true,
	}
}

func toResult(ctx *extern.Context, result values.BalValue, err error) (values.BalValue, error) {
	if err == nil {
		return result, nil
	}
	handle, ok := ctx.LookupFunction(orgName, moduleName, "createError")
	if !ok {
		return nil, fmt.Errorf("data.xmldata: internal helper function createError not found")
	}
	return ctx.InvokeFunction(handle, []values.BalValue{err.Error()})
}
