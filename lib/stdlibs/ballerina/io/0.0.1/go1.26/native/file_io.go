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

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

type fileIOTypes struct {
	strArrTy   semtypes.SemType
	byteArrTy  semtypes.SemType
	jsonListTy semtypes.SemType
	jsonMapTy  semtypes.SemType
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

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileReadString",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			path, _ := args[0].(string)
			data, err := rt.Platform().FS.ReadFile(path)
			if err != nil {
				return fileIOError(fmt.Sprintf("error while reading file '%s': %s", path, err.Error())), nil
			}
			return strings.Join(splitLines(data), "\n"), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileReadLines",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {

			path, _ := args[0].(string)
			data, err := rt.Platform().FS.ReadFile(path)
			if err != nil {
				return fileIOError(fmt.Sprintf("error while reading file '%s': %s", path, err.Error())), nil
			}
			lines := splitLines(data)
			items := make([]values.BalValue, len(lines))
			for i, line := range lines {
				items[i] = line
			}
			return values.NewList(types.strArrTy, semtypes.ToListAtomicType(ctx.TypeCtx, types.strArrTy), false, nil, 0, items), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileReadBytes",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {

			path, _ := args[0].(string)
			data, err := rt.Platform().FS.ReadFile(path)
			if err != nil {
				return fileIOError(fmt.Sprintf("error while reading file '%s': %s", path, err.Error())), nil
			}
			items := make([]values.BalValue, len(data))
			for i, b := range data {
				items[i] = int64(b)
			}
			return values.NewList(types.byteArrTy, semtypes.ToListAtomicType(ctx.TypeCtx, types.byteArrTy), false, nil, 0, items), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileReadJson",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {

			path, _ := args[0].(string)
			data, err := rt.Platform().FS.ReadFile(path)
			if err != nil {
				return fileIOError(fmt.Sprintf("error while reading file '%s': %s", path, err.Error())), nil
			}
			dec := json.NewDecoder(strings.NewReader(string(data)))
			raw, err := values.DecodeJSON(dec, ctx.TypeCtx, types.jsonListTy, types.jsonMapTy)
			if err != nil {
				return fileIOError(fmt.Sprintf("error while parsing JSON from file '%s': %s", path, err.Error())), nil
			}
			var extra any
			if err := dec.Decode(&extra); err != io.EOF {
				if err == nil {
					return fileIOError(fmt.Sprintf("trailing content after JSON value in file '%s'", path)), nil
				}
				return fileIOError(fmt.Sprintf("error reading trailing content in file '%s': %s", path, err.Error())), nil
			}
			return raw, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileWriteString",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			path, _ := args[0].(string)
			content, _ := args[1].(string)
			option, _ := args[2].(string)
			data := []byte(content)
			var err error
			if option == "APPEND" {
				err = rt.Platform().FS.AppendFile(path, data)
			} else {
				err = rt.Platform().FS.WriteFile(path, data)
			}
			if err != nil {
				return fileIOError(fmt.Sprintf("error while writing to file '%s': %s", path, err.Error())), nil
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileWriteLines",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
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
				err = rt.Platform().FS.AppendFile(path, data)
			} else {
				err = rt.Platform().FS.WriteFile(path, data)
			}
			if err != nil {
				return fileIOError(fmt.Sprintf("error while writing to file '%s': %s", path, err.Error())), nil
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileWriteBytes",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			path, _ := args[0].(string)
			list, _ := args[1].(*values.List)
			option, _ := args[2].(string)
			data := list.ToByteSlice()
			var err error
			if option == "APPEND" {
				err = rt.Platform().FS.AppendFile(path, data)
			} else {
				err = rt.Platform().FS.WriteFile(path, data)
			}
			if err != nil {
				return fileIOError(fmt.Sprintf("error while writing to file '%s': %s", path, err.Error())), nil
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileWriteJson",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			path, _ := args[0].(string)
			data, err := values.ToJSONByteArray(args[1])
			if err != nil {
				return fileIOError(fmt.Sprintf("error while serializing JSON for file '%s': %s", path, err.Error())), nil
			}
			if err := rt.Platform().FS.WriteFile(path, data); err != nil {
				return fileIOError(fmt.Sprintf("error while writing to file '%s': %s", path, err.Error())), nil
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileReadXml",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {

			path, _ := args[0].(string)
			data, err := rt.Platform().FS.ReadFile(path)
			if err != nil {
				return fileIOError(fmt.Sprintf("error while reading file '%s': %s", path, err.Error())), nil
			}
			xmlVal, parseErr := values.ParseAsXMLValue(ctx.TypeCtx, values.FromBytes(data), values.XMLLenientMode)
			if parseErr != nil {
				return fileIOError(fmt.Sprintf("error while parsing XML from file '%s': %s", path, parseErr.Error())), nil
			}
			return xmlVal, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileWriteXml",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			path, _ := args[0].(string)
			content, _ := args[1].(values.XMLValue)
			option, _ := args[2].(string)
			data := []byte(content.XMLString())
			var err error
			if option == "APPEND" {
				err = rt.Platform().FS.AppendFile(path, data)
			} else {
				err = rt.Platform().FS.WriteFile(path, data)
			}
			if err != nil {
				return fileIOError(fmt.Sprintf("error while writing to file '%s': %s", path, err.Error())), nil
			}
			return nil, nil
		})
}

func init() {
	runtime.RegisterModuleInitializer(initFileIOModule)
}
