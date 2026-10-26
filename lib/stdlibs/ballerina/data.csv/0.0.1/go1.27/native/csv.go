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
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "data.csv"

	streamReadErrorPrefix = "Error occurred while reading the stream: "
)

type csvModule struct {
	rt    *runtime.Runtime
	types *csvTypes
}

func init() {
	runtime.RegisterModuleInitializer(initCsvModule)
}

func initCsvModule(rt *runtime.Runtime) {
	m := &csvModule{rt: rt, types: newCsvTypes(rt.GetTypeEnv())}
	runtime.RegisterExternFunction(rt, orgName, moduleName, "parseString", m.parseString)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "parseBytes", m.parseBytes)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "parseStream", m.parseStream)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "transform", m.transform)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "parseList", m.parseList)
}

// newError creates a csv:Error, which is a plain alias of error.
func newError(message string) *values.Error {
	return values.NewError(semtypes.Error, message, nil, "Error", nil)
}

func toResult(result values.BalValue, err error) (values.BalValue, error) {
	if convErr, ok := err.(*conversionError); ok {
		return convErr.value, nil
	}
	if err != nil {
		return newError(err.Error()), nil
	}
	return result, nil
}

// targetType is the type a typedesc argument describes. An inferred typedesc can carry the error part
// of the call's return type, which is dropped.
func targetType(td *values.TypeDesc) semtypes.SemType {
	return semtypes.Diff(td.Constraint(), semtypes.Error)
}

func (m *csvModule) parseString(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	input, _ := args[0].(string)
	opts, _ := args[1].(*values.Map)
	td, _ := args[2].(*values.TypeDesc)
	cfg := readOptions(opts)
	t := newTyper(ctx.TypeCtx(), m.types)
	result, err := m.parseInput(cfg, t, targetType(td), input, input)
	return toResult(result, err)
}

func (m *csvModule) parseBytes(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	bytes, _ := args[0].(*values.List)
	opts, _ := args[1].(*values.Map)
	td, _ := args[2].(*values.TypeDesc)
	cfg := readOptions(opts)
	t := newTyper(ctx.TypeCtx(), m.types)
	root := targetType(td)
	dec, ok := newDecoder(cfg.encoding)
	if !ok {
		return newError(invalidCastError(javaValueOf(bytes), "typedesc "+t.typeName(root)).Error()), nil
	}
	input := dec.decodeAll(bytes.ToByteSlice())
	result, err := m.parseInput(cfg, t, root, input, javaValueOf(bytes))
	return toResult(result, err)
}

func (m *csvModule) parseStream(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	source, _ := args[0].(*values.Stream)
	opts, _ := args[1].(*values.Map)
	td, _ := args[2].(*values.TypeDesc)
	cfg := readOptions(opts)
	dec, ok := newDecoder(cfg.encoding)
	if !ok {
		return newError(streamReadErrorPrefix + cfg.encoding), nil
	}
	content, sourceErr := readAllBlocks(source)
	if sourceErr != nil {
		return sourceErr, nil
	}
	input := dec.decodeAll(content)
	t := newTyper(ctx.TypeCtx(), m.types)
	result, err := m.parseInput(cfg, t, targetType(td), input, input)
	if err != nil {
		return newError(streamReadErrorPrefix + err.Error()), nil
	}
	return result, nil
}

func (m *csvModule) transform(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	source, _ := args[0].(*values.List)
	opts, _ := args[1].(*values.Map)
	td, _ := args[2].(*values.TypeDesc)
	cfg := readOptions(opts)
	return m.traverseList(ctx, cfg, source, td)
}

func (m *csvModule) parseList(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	source, _ := args[0].(*values.List)
	opts, _ := args[1].(*values.Map)
	td, _ := args[2].(*values.TypeDesc)
	cfg := readOptions(opts)
	cfg.stringConversion = true
	return m.traverseList(ctx, cfg, source, td)
}

func (m *csvModule) traverseList(ctx *extern.Context, cfg *config, source *values.List,
	td *values.TypeDesc) (values.BalValue, error) {
	rows := make([]values.BalValue, source.Len())
	for i := range rows {
		rows[i] = source.Get(i)
	}
	t := newTyper(ctx.TypeCtx(), m.types)
	tr := newTraversal(cfg, t, newFailSafeLogger(m.rt, cfg.failSafe, false))
	return toResult(tr.traverse(rows, targetType(td)))
}

// parseInput parses CSV text into root, a list type or a union of list types. sourceText is how the
// input appears in the error jBallerina reports when no member of a union element type fits.
func (m *csvModule) parseInput(cfg *config, t *typer, root semtypes.SemType, input, sourceText string) (values.BalValue, error) {
	rootInfo := t.classify(root)
	switch rootInfo.kind {
	case kindUnion:
		return m.parseIntoUnion(cfg, t, root, input, sourceText)
	case kindArray:
	default:
		return nil, invalidTypeError(t.typeName(root))
	}
	rootShape := t.rowShapeOf(rootInfo)
	elemInfo := t.classify(rootShape.rest)
	elem := t.rowShapeOf(elemInfo)
	switch {
	case elem.kind == kindUnion:
		return m.parseIntoUnion(cfg, t, root, input, sourceText)
	case !elem.isMapping() && !elem.isList():
		return nil, sourceCannotConvertError(t.typeName(rootShape.rest))
	}
	p := newParser(cfg, t, &elem, rootShape.size, newFailSafeLogger(m.rt, cfg.failSafe, true))
	if err := feedAll(p, input); err != nil {
		return nil, err
	}
	return buildList(t, root, rootInfo.list, rootShape.readonly, p.rows)
}

// parseIntoUnion reads the rows as map<string> values and converts them as a whole to the first member
// type that every row fits, as jBallerina does for union types.
func (m *csvModule) parseIntoUnion(cfg *config, t *typer, root semtypes.SemType, input, sourceText string) (values.BalValue, error) {
	parseLogger := newFailSafeLogger(m.rt, cfg.failSafe, true)
	rows, err := m.parseAsStringMaps(cfg, t, input, parseLogger)
	if err != nil {
		return nil, err
	}
	traversalCfg := *cfg
	traversalCfg.stringConversion = true
	traversalCfg.headerRows = 0
	traversalCfg.customHeaders = nil
	traversalCfg.headerOrder = nil
	result, err := newTraversal(&traversalCfg, t, parseLogger.withoutConsole()).traverse(rows, root)
	if err != nil && cfg.enableConstraintValidation {
		// jBallerina hands the traversal's error value to constraint validation, which fails on it with a
		// non-csv exception that the entry point reports as this generic cast error.
		return nil, invalidCastError(sourceText, "typedesc "+t.typeName(root))
	}
	return result, err
}

func (m *csvModule) parseAsStringMaps(cfg *config, t *typer, input string, logger *failSafeLogger) ([]values.BalValue, error) {
	shape := t.rowShapeOf(t.classify(t.stringMapTy))
	memberCfg := cfg.unionMemberConfig()
	p := newParser(memberCfg, t, &shape, -1, logger)
	if err := feedAll(p, input); err != nil {
		return nil, err
	}
	return p.rows, nil
}

func feedAll(p *parser, input string) error {
	for _, ch := range input {
		if err := p.feed(ch); err != nil {
			return err
		}
	}
	return p.finish()
}
