// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

type fileIOTypes struct {
	strArrTy      semtypes.SemType
	byteArrTy     semtypes.SemType
	byteArrAtom   *semtypes.ListAtomicType
	roByteArrTy   semtypes.SemType
	roByteArrAtom *semtypes.ListAtomicType
	jsonListTy    semtypes.SemType
	jsonMapTy     semtypes.SemType

	lineStreamTy   semtypes.SemType
	lineRecordTy   semtypes.SemType
	lineRecordAtom *semtypes.MappingAtomicType

	blockStreamTy   semtypes.SemType
	blockRecordTy   semtypes.SemType
	blockRecordAtom *semtypes.MappingAtomicType
}

// closedNextRecordType builds the `record {| T value; |}` type used as the
// per-element value yielded by a stream's `next()` method.
func closedNextRecordType(env semtypes.Env, valueTy semtypes.SemType) semtypes.SemType {
	md := semtypes.NewMappingDefinition()
	return md.DefineMappingTypeWrapped(env, []semtypes.Field{semtypes.FieldFrom("value", valueTy, false, false)}, semtypes.NEVER)
}

func fileIOError(msg string) values.BalValue {
	return values.NewErrorWithMessage(msg)
}

func splitLines(data []byte) []string {
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// fileIOExterns implements the whole-file I/O extern functions; each is
// registered as a named method rather than an inline closure.
type fileIOExterns struct {
	rt    *runtime.Runtime
	types fileIOTypes
}

func (e *fileIOExterns) readString(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	path, _ := args[0].(string)
	data, err := e.rt.Platform().FS.ReadFile(path)
	if err != nil {
		return fileIOError(fmt.Sprintf("error while reading file '%s': %s", path, err.Error())), nil
	}
	return strings.Join(splitLines(data), "\n"), nil
}

func (e *fileIOExterns) readLines(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	path, _ := args[0].(string)
	data, err := e.rt.Platform().FS.ReadFile(path)
	if err != nil {
		return fileIOError(fmt.Sprintf("error while reading file '%s': %s", path, err.Error())), nil
	}
	lines := splitLines(data)
	items := make([]values.BalValue, len(lines))
	for i, line := range lines {
		items[i] = line
	}
	return values.NewList(e.types.strArrTy, semtypes.ToListAtomicType(ctx.TypeEnv(), e.types.strArrTy), false, nil, 0, items), nil
}

func (e *fileIOExterns) readBytes(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	path, _ := args[0].(string)
	data, err := e.rt.Platform().FS.ReadFile(path)
	if err != nil {
		return fileIOError(fmt.Sprintf("error while reading file '%s': %s", path, err.Error())), nil
	}
	items := make([]values.BalValue, len(data))
	for i, b := range data {
		items[i] = int64(b)
	}
	return values.NewList(e.types.roByteArrTy, e.types.roByteArrAtom, true, nil, 0, items), nil
}

func (e *fileIOExterns) readJson(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	path, _ := args[0].(string)
	data, err := e.rt.Platform().FS.ReadFile(path)
	if err != nil {
		return fileIOError(fmt.Sprintf("error while reading file '%s': %s", path, err.Error())), nil
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	var raw any
	if err := dec.Decode(&raw); err != nil {
		return fileIOError(fmt.Sprintf("error while parsing JSON from file '%s': %s", path, err.Error())), nil
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fileIOError(fmt.Sprintf("trailing content after JSON value in file '%s'", path)), nil
		}
		return fileIOError(fmt.Sprintf("error reading trailing content in file '%s': %s", path, err.Error())), nil
	}
	return values.GoToBalValue(ctx.TypeCtx(), raw, e.types.jsonListTy, e.types.jsonMapTy), nil
}

func (e *fileIOExterns) writeString(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	path, _ := args[0].(string)
	content, _ := args[1].(string)
	option, _ := args[2].(string)
	data := []byte(content)
	var err error
	if option == "APPEND" {
		err = e.rt.Platform().FS.AppendFile(path, data)
	} else {
		err = e.rt.Platform().FS.WriteFile(path, data)
	}
	if err != nil {
		return fileIOError(fmt.Sprintf("error while writing to file '%s': %s", path, err.Error())), nil
	}
	return nil, nil
}

func (e *fileIOExterns) writeLines(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	path, _ := args[0].(string)
	list, _ := args[1].(*values.List)
	option, _ := args[2].(string)
	var sb strings.Builder
	for i := range list.Len() {
		line, _ := list.Get(i).(string)
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	data := []byte(sb.String())
	var err error
	if option == "APPEND" {
		err = e.rt.Platform().FS.AppendFile(path, data)
	} else {
		err = e.rt.Platform().FS.WriteFile(path, data)
	}
	if err != nil {
		return fileIOError(fmt.Sprintf("error while writing to file '%s': %s", path, err.Error())), nil
	}
	return nil, nil
}

func (e *fileIOExterns) writeBytes(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	path, _ := args[0].(string)
	list, _ := args[1].(*values.List)
	option, _ := args[2].(string)
	data := list.ToByteSlice()
	var err error
	if option == "APPEND" {
		err = e.rt.Platform().FS.AppendFile(path, data)
	} else {
		err = e.rt.Platform().FS.WriteFile(path, data)
	}
	if err != nil {
		return fileIOError(fmt.Sprintf("error while writing to file '%s': %s", path, err.Error())), nil
	}
	return nil, nil
}

func (e *fileIOExterns) writeJson(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	path, _ := args[0].(string)
	data, err := values.ToJSONByteArray(args[1])
	if err != nil {
		return fileIOError(fmt.Sprintf("error while serializing JSON for file '%s': %s", path, err.Error())), nil
	}
	if err := e.rt.Platform().FS.WriteFile(path, data); err != nil {
		return fileIOError(fmt.Sprintf("error while writing to file '%s': %s", path, err.Error())), nil
	}
	return nil, nil
}

func (e *fileIOExterns) readXml(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	path, _ := args[0].(string)
	data, err := e.rt.Platform().FS.ReadFile(path)
	if err != nil {
		return fileIOError(fmt.Sprintf("error while reading file '%s': %s", path, err.Error())), nil
	}
	xmlVal, parseErr := values.ParseAsXMLValue(ctx.TypeCtx(), values.FromBytes(data), values.XMLLenientMode)
	if parseErr != nil {
		return fileIOError(fmt.Sprintf("error while parsing XML from file '%s': %s", path, parseErr.Error())), nil
	}
	return xmlVal, nil
}

func (e *fileIOExterns) writeXml(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	path, _ := args[0].(string)
	content, _ := args[1].(values.XMLValue)
	option, _ := args[2].(string)
	data := []byte(content.XMLString())
	var err error
	if option == "APPEND" {
		err = e.rt.Platform().FS.AppendFile(path, data)
	} else {
		err = e.rt.Platform().FS.WriteFile(path, data)
	}
	if err != nil {
		return fileIOError(fmt.Sprintf("error while writing to file '%s': %s", path, err.Error())), nil
	}
	return nil, nil
}

func initFileIOModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	typCtx := semtypes.ContextFrom(env)
	jsonTy := semtypes.CreateJSON(typCtx)
	sld := semtypes.NewListDefinition()
	bld := semtypes.NewListDefinition()
	jmd := semtypes.NewMappingDefinition()
	jld := semtypes.NewListDefinition()
	types := fileIOTypes{
		strArrTy:   sld.DefineListTypeWrappedWithEnvSemType(env, semtypes.STRING),
		byteArrTy:  bld.DefineListTypeWrappedWithEnvSemType(env, semtypes.BYTE),
		jsonMapTy:  jmd.DefineMappingTypeWrapped(env, nil, jsonTy),
		jsonListTy: jld.DefineListTypeWrappedWithEnvSemType(env, jsonTy),
	}

	types.byteArrAtom = semtypes.ToListAtomicType(env, types.byteArrTy)
	// io:Block is `readonly & byte[]`; a CELL_MUT_NONE list definition is the
	// atom-backed equivalent of that intersection.
	robld := semtypes.NewListDefinition()
	types.roByteArrTy = robld.DefineListTypeWrappedWithEnvSemTypeCellMutability(env, semtypes.BYTE, semtypes.CellMutability_CELL_MUT_NONE)
	types.roByteArrAtom = semtypes.ToListAtomicType(env, types.roByteArrTy)

	streamCompletionTy := semtypes.Union(semtypes.ERROR, semtypes.NIL)

	lsd := semtypes.NewStreamDefinition()
	types.lineRecordTy = closedNextRecordType(env, semtypes.STRING)
	types.lineRecordAtom = semtypes.ToMappingAtomicType(typCtx, types.lineRecordTy)
	types.lineStreamTy = lsd.Define(env, semtypes.STRING, streamCompletionTy)

	bsd := semtypes.NewStreamDefinition()
	types.blockRecordTy = closedNextRecordType(env, types.roByteArrTy)
	types.blockRecordAtom = semtypes.ToMappingAtomicType(typCtx, types.blockRecordTy)
	types.blockStreamTy = bsd.Define(env, types.roByteArrTy, streamCompletionTy)

	e := &fileIOExterns{rt: rt, types: types}
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileReadString", e.readString)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileReadLines", e.readLines)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileReadBytes", e.readBytes)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileReadJson", e.readJson)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileWriteString", e.writeString)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileWriteLines", e.writeLines)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileWriteBytes", e.writeBytes)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileWriteJson", e.writeJson)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileReadXml", e.readXml)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileWriteXml", e.writeXml)

	registerStreamIOExterns(rt, types)
}

func init() {
	runtime.RegisterModuleInitializer(initFileIOModule)
}
