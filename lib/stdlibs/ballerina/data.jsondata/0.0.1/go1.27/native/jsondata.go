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
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "data.jsondata"
)

func init() {
	runtime.RegisterModuleInitializer(initJsondataModule)
}

func initJsondataModule(rt *runtime.Runtime) {
	cache := newTypeCache()
	runtime.RegisterExternFunction(rt, orgName, moduleName, "parseAsType", withTypeCache(cache, parseAsType))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "parseString", withTypeCache(cache, parseString))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "parseBytes", withTypeCache(cache, parseBytes))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "parseStream", withTypeCache(cache, parseStream))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externToJson", withTypeCache(cache, toJson))
}

func withTypeCache(cache *typeCache, impl func(*typeCache, *extern.Context, []values.BalValue) (values.BalValue, error)) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return impl(cache, ctx, args)
	}
}

func parseAsType(cache *typeCache, ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	opts, _ := args[1].(*values.Map)
	td, _ := args[2].(*values.TypeDesc)
	t := newTyperFor(cache, ctx, td)
	result, err := traverse(t, readProjection(opts), args[0], t.root)
	return toResult(ctx, result, err)
}

func parseString(cache *typeCache, ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	input, _ := args[0].(string)
	return parseText(cache, ctx, utf16FromString(input), args[1], args[2])
}

func parseBytes(cache *typeCache, ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	input, _ := args[0].(*values.List)
	return parseUTF8(cache, ctx, input.ToByteSlice(), args[1], args[2])
}

// parseStream reads the whole byte-block stream before parsing. Unlike jBallerina, which treats an error from
// the stream as the end of the input, an error from the stream is returned as a jsondata:Error.
func parseStream(cache *typeCache, ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	byteStream, _ := args[0].(*values.Stream)
	var input []byte
	for {
		next := byteStream.Next()
		if streamErr, ok := next.(*values.Error); ok {
			return streamError(ctx, streamErr)
		}
		record, ok := next.(*values.Map)
		if !ok {
			break
		}
		block, _ := record.Get("value")
		input = append(input, block.(*values.List).ToByteSlice()...)
	}
	return parseUTF8(cache, ctx, input, args[1], args[2])
}

func streamError(ctx *extern.Context, cause *values.Error) (values.BalValue, error) {
	result, err := toResult(ctx, nil, &jsonError{message: cause.Message})
	if jsonErr, ok := result.(*values.Error); ok {
		jsonErr.Cause = cause
	}
	return result, err
}

func parseUTF8(cache *typeCache, ctx *extern.Context, input []byte, optsArg, tdArg values.BalValue) (values.BalValue, error) {
	text, err := utf16FromBytes(input)
	if err != nil {
		return toResult(ctx, nil, err)
	}
	return parseText(cache, ctx, text, optsArg, tdArg)
}

func parseText(cache *typeCache, ctx *extern.Context, input []uint16, optsArg, tdArg values.BalValue) (values.BalValue, error) {
	opts, _ := optsArg.(*values.Map)
	td, _ := tdArg.(*values.TypeDesc)
	t := newTyperFor(cache, ctx, td)
	result, err := parse(t, readProjection(opts), input, t.root)
	return toResult(ctx, result, err)
}

func toJson(cache *typeCache, ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	t := newTyper(cache, ctx.TypeCtx(), semtypes.Never, nil)
	result, err := t.toJSON(args[0], make(map[values.BalValue]struct{}))
	return toResult(ctx, result, err)
}

// newTyperFor drops error from the typedesc constraint: an inferred typedesc can wrongly carry the module's
// distinct error type (ballerina-nutcracker/ballerina#1383).
func newTyperFor(cache *typeCache, ctx *extern.Context, td *values.TypeDesc) *typer {
	return newTyper(cache, ctx.TypeCtx(), semtypes.Diff(td.Constraint(), semtypes.Error), fieldKeys(ctx, td))
}

const nameAnnotationKey = "ballerina/data.jsondata:0.0.1:Name"

func fieldKeys(ctx *extern.Context, td *values.TypeDesc) map[string]string {
	annotations, _ := ctx.TypeAnnotations(td)
	keys := make(map[string]string)
	for field, fieldAnnotations := range annotations.Fields {
		config, ok := fieldAnnotations[nameAnnotationKey].(*values.Map)
		if !ok {
			continue
		}
		if name, ok := config.Get("value"); ok {
			keys[field] = fmt.Sprint(name)
		}
	}
	return keys
}

func readProjection(opts *values.Map) projection {
	allow, _ := opts.Get("allowDataProjection")
	config, ok := allow.(*values.Map)
	if !ok {
		return projection{}
	}
	nilAsOptional, _ := config.Get("nilAsOptionalField")
	absentAsNilable, _ := config.Get("absentAsNilableType")
	return projection{
		allowDataProjection: true,
		nilAsOptionalField:  nilAsOptional == true,
		absentAsNilableType: absentAsNilable == true,
	}
}

func toResult(ctx *extern.Context, result values.BalValue, err error) (values.BalValue, error) {
	if err == nil {
		return result, nil
	}
	handle, ok := ctx.LookupFunction(orgName, moduleName, "createError")
	if !ok {
		return nil, fmt.Errorf("data.jsondata: internal helper function createError not found")
	}
	return ctx.InvokeFunction(handle, []values.BalValue{err.Error()})
}
