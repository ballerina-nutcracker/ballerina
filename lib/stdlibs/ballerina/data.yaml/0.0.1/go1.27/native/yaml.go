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
	"strings"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "data.yaml"

	streamReadErrorPrefix = "Error occurred while reading the stream: "
)

type yamlSchema int

const (
	schemaFailsafe yamlSchema = iota
	schemaJSON
	schemaCore
)

func schemaOf(v values.BalValue) yamlSchema {
	switch v {
	case "FAILSAFE_SCHEMA":
		return schemaFailsafe
	case "JSON_SCHEMA":
		return schemaJSON
	}
	return schemaCore
}

type readOptions struct {
	schema                    yamlSchema
	allowAnchorRedefinition   bool
	allowMapEntryRedefinition bool
	allowDataProjection       bool
	nilAsOptionalField        bool
	absentAsNilableType       bool
	enableYamlStreamReorder   bool
}

func readReadOptions(opts *values.Map) *readOptions {
	options := &readOptions{
		schema:                    schemaOf(get(opts, "schema")),
		allowAnchorRedefinition:   get(opts, "allowAnchorRedefinition") != false,
		allowMapEntryRedefinition: get(opts, "allowMapEntryRedefinition") == true,
	}
	if projection, ok := get(opts, "allowDataProjection").(*values.Map); ok {
		options.allowDataProjection = true
		options.nilAsOptionalField = get(projection, "nilAsOptionalField") == true
		options.absentAsNilableType = get(projection, "absentAsNilableType") == true
		options.enableYamlStreamReorder = get(projection, "enableYamlStreamReorder") == true
	}
	return options
}

func get(m *values.Map, key string) values.BalValue {
	if m == nil {
		return nil
	}
	v, _ := m.Get(key)
	return v
}

func init() {
	runtime.RegisterModuleInitializer(initYamlModule)
}

func initYamlModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "parseString", parseString)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "parseBytes", parseBytes)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "parseStream", parseStream)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toYamlString", toYamlString)
}

func parseString(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	input, _ := args[0].(string)
	return parseInput(ctx, input, args[1], args[2], "")
}

func parseBytes(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	input, _ := args[0].(*values.List)
	return parseInput(ctx, decodeUTF8(input.ToByteSlice()), args[1], args[2], "")
}

// parseStream reads the whole byte-block stream before parsing. An error from the stream is returned
// rather than ending the input early, as jBallerina does.
func parseStream(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	source, _ := args[0].(*values.Stream)
	var content []byte
	for {
		next := source.Next()
		if sourceErr, isErr := next.(*values.Error); isErr {
			closeSource(source)
			return createError(ctx, streamReadErrorPrefix+sourceErr.Message)
		}
		record, ok := next.(*values.Map)
		if !ok {
			break
		}
		block, _ := record.Get("value")
		list, _ := block.(*values.List)
		content = append(content, list.ToByteSlice()...)
	}
	closeSource(source)
	return parseInput(ctx, decodeUTF8(content), args[1], args[2], streamReadErrorPrefix)
}

func closeSource(source *values.Stream) {
	if source.Close != nil {
		source.Close()
	}
}

// decodeUTF8 decodes bytes as UTF-8, replacing malformed sequences with U+FFFD, as Java's default
// reader does. A leading byte-order mark is dropped.
func decodeUTF8(data []byte) string {
	return strings.TrimPrefix(strings.ToValidUTF8(string(data), "�"), "\uFEFF")
}

// parseInput parses input into the type the typedesc argument describes. errorPrefix is added to the
// errors that jBallerina's parseStream wraps: those raised while binding values, not syntax errors.
func parseInput(ctx *extern.Context, input string, optsArg, tdArg values.BalValue, errorPrefix string) (values.BalValue, error) {
	opts, _ := optsArg.(*values.Map)
	td, _ := tdArg.(*values.TypeDesc)
	t := newTyper(ctx.TypeCtx())
	root := semtypes.Diff(td.Constraint(), semtypes.Error)
	options := readReadOptions(opts)
	c := &converter{t: t, options: options, root: root, renames: fieldRenames(ctx, td)}
	result, err := compose(c, strings.TrimPrefix(input, "\uFEFF"), root)
	if err == nil {
		return result, nil
	}
	message := err.Error()
	if yamlErr, ok := err.(*yamlError); !ok || yamlErr.noPosition {
		message = errorPrefix + message
	}
	return createError(ctx, message)
}

// fieldRenames maps YAML keys to field names for the fields of the target record that carry a Name
// annotation. As in jBallerina, the first annotation whose name ends with "Name" supplies the key. The
// annotations are only available when the typedesc names the record type itself.
func fieldRenames(ctx *extern.Context, td *values.TypeDesc) map[string]string {
	annotations, _ := ctx.TypeAnnotations(td)
	renames := make(map[string]string)
	for field, fieldAnnotations := range annotations.Fields {
		for key, value := range fieldAnnotations {
			config, ok := value.(*values.Map)
			if !ok || !strings.HasSuffix(key, "Name") {
				continue
			}
			if name, ok := config.Get("value"); ok {
				renames[fmt.Sprint(name)] = field
				break
			}
		}
	}
	return renames
}

func createError(ctx *extern.Context, message string) (values.BalValue, error) {
	handle, ok := ctx.LookupFunction(orgName, moduleName, "createError")
	if !ok {
		return nil, fmt.Errorf("data.yaml: internal helper function createError not found")
	}
	return ctx.InvokeFunction(handle, []values.BalValue{message})
}

func compose(c *converter, input string, root semtypes.SemType) (values.BalValue, error) {
	t, options := c.t, c.options
	rootInfo := t.classify(root)
	if rootInfo.kind == kindUnsupported {
		return nil, newConversionError("unsupported type '%s'", t.typeName(root))
	}
	if input == "" {
		return nil, &yamlError{message: "invalid yaml document", line: 1, column: 0}
	}
	p := newParser(input, options)
	if !isStreamTarget(t, rootInfo) {
		doc, _, ok, err := p.nextDocument()
		if err != nil {
			return nil, err
		}
		if !ok {
			doc = &node{kind: nodeScalar, empty: true}
		}
		return c.convert(doc, root, scope{})
	}
	var docs []*node
	for {
		doc, _, ok, err := p.nextDocument()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		docs = append(docs, doc)
	}
	if len(docs) == 0 && p.sawDocumentEnd {
		docs = append(docs, &node{kind: nodeScalar, empty: true})
	}
	if len(docs) == 1 {
		return c.convert(docs[0], root, scope{})
	}
	if options.enableYamlStreamReorder && rootInfo.kind == kindTuple {
		return c.reorderDocuments(docs, rootInfo)
	}
	return c.convert(&node{kind: nodeSequence, items: docs}, root, scope{})
}

// isStreamTarget reports whether several documents can bind to the type: a list, json, anydata, or a
// union with such a member. Any other type binds to the first document only.
func isStreamTarget(t *typer, ti typeInfo) bool {
	switch ti.kind {
	case kindArray, kindTuple, kindJSON, kindAnydata:
		return true
	case kindUnion:
		for _, member := range ti.members {
			if isStreamTarget(t, t.classify(member)) {
				return true
			}
		}
	}
	return false
}

// reorderDocuments fills a tuple from documents in any order: each document goes to the first unfilled
// member it converts to, and otherwise to the rest type.
func (c *converter) reorderDocuments(docs []*node, tuple typeInfo) (values.BalValue, error) {
	shape := c.t.rowShapeOf(tuple)
	filled := make([]values.BalValue, shape.size)
	unfilled := make([]int, shape.size)
	for i := range unfilled {
		unfilled[i] = i
	}
	var rest []values.BalValue
	for _, doc := range docs {
		if len(unfilled) == 0 && !shape.hasRest {
			break
		}
		matched := false
		for i, index := range unfilled {
			memberType, _ := shape.memberType(index)
			if value, err := c.convert(doc, memberType, scope{}); err == nil {
				filled[index] = value
				unfilled = append(unfilled[:i], unfilled[i+1:]...)
				matched = true
				break
			}
		}
		if !matched && shape.hasRest {
			if value, err := c.convert(doc, shape.rest, scope{}); err == nil {
				rest = append(rest, value)
			}
		}
	}
	if len(unfilled) > 0 {
		return nil, arraySizeError()
	}
	return buildList(c.t, shape.ty, shape.list, shape.readonly, append(filled, rest...))
}
