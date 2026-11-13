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
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "zip"
)

type errorKind string

const (
	invalidArchive   errorKind = "InvalidArchiveError"
	entryNotFound    errorKind = "EntryNotFoundError"
	unsupportedEntry errorKind = "UnsupportedEntryError"
	fileSystem       errorKind = "FileSystemError"
)

type zipTypes struct {
	byteArrTy   semtypes.SemType
	byteArrAtom *semtypes.ListAtomicType
	chunkTy     semtypes.SemType
	chunkAtom   *semtypes.MappingAtomicType
}

// zipExterns implements the extern functions of the module; each is registered as a named method.
type zipExterns struct {
	rt    *runtime.Runtime
	types zipTypes
}

func invokeHelper(ctx *extern.Context, name string, args ...values.BalValue) (values.BalValue, error) {
	handle, ok := ctx.LookupFunction(orgName, moduleName, name)
	if !ok {
		return nil, fmt.Errorf("zip: internal helper function %s not found", name)
	}
	return ctx.InvokeFunction(handle, args)
}

// zipError builds an error of the module through the .bal nativeError helper, since only Ballerina can
// construct a value of a distinct error type.
func zipError(ctx *extern.Context, kind errorKind, message string) (values.BalValue, error) {
	return invokeHelper(ctx, "nativeError", string(kind), message, "")
}

func entryError(ctx *extern.Context, kind errorKind, message, entryName string) (values.BalValue, error) {
	return invokeHelper(ctx, "nativeError", string(kind), message, entryName)
}

func (e *zipExterns) newByteArray(data []byte) *values.List {
	items := make([]values.BalValue, len(data))
	for i, b := range data {
		items[i] = int64(b)
	}
	return values.NewList(e.types.byteArrTy, e.types.byteArrAtom, false, nil, 0, items)
}

func initZipModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	ld := semtypes.NewListDefinition()
	types := zipTypes{byteArrTy: ld.Define(env, nil, semtypes.ListRest(semtypes.Byte))}
	types.byteArrAtom = semtypes.ToListAtomicType(env, types.byteArrTy)
	md := semtypes.NewMappingDefinition()
	types.chunkTy = md.Define(env, []semtypes.Field{semtypes.FieldFrom("value", types.byteArrTy, false, false)},
		semtypes.Never)
	types.chunkAtom = semtypes.ToMappingAtomicType(semtypes.ContextFrom(env), types.chunkTy)

	e := &zipExterns{rt: rt, types: types}
	externs := map[string]extern.NativeFunc{
		"nativeOpen":              e.open,
		"nativeEntries":           e.entries,
		"nativeIndexOf":           e.indexOf,
		"nativeEntryAt":           e.entryAt,
		"nativeExtractEntry":      e.extractEntry,
		"nativeCloseArchive":      e.closeArchive,
		"ArchiveReader.readEntry": e.readEntry,
		"nativeCreate":            e.create,
		"nativeAddFile":           e.addFile,
		"nativeStartEntry":        e.startEntry,
		"nativeWriteChunk":        e.writeChunk,
		"nativeFinishEntry":       e.finishEntry,
		"nativeCopyEntry":         e.copyEntry,
		"nativeCloseWriter":       e.closeWriter,
		"nativeRealPath":          e.realPath,
		"nativeSetModifiedTime":   e.setModifiedTime,
		"nativeIsSameFile":        e.isSameFile,
	}
	for name, impl := range externs {
		runtime.RegisterExternFunction(rt, orgName, moduleName, name, impl)
	}
}

func init() {
	runtime.RegisterModuleInitializer(initZipModule)
}
