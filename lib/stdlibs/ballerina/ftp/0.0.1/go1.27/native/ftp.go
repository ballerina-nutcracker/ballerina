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
	"path"
	goruntime "runtime"
	"strings"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "ftp"
)

// lineSeparator stands in for Java's System.lineSeparator(), which CSVUtils writes after rows.
var lineSeparator = func() string {
	if goruntime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}()

type ftpTypes struct {
	env            semtypes.Env
	jsonTy         semtypes.SemType
	jsonListTy     semtypes.SemType
	jsonListAtom   *semtypes.ListAtomicType
	jsonMapTy      semtypes.SemType
	jsonMapAtom    *semtypes.MappingAtomicType
	byteArrTy      semtypes.SemType
	byteArrAtom    *semtypes.ListAtomicType
	roByteArrTy    semtypes.SemType
	roByteArrAtom  *semtypes.ListAtomicType
	strArrTy       semtypes.SemType
	strArrAtom     *semtypes.ListAtomicType
	strArrArrTy    semtypes.SemType
	strArrArrAtom  *semtypes.ListAtomicType
	byteRecordTy   semtypes.SemType
	byteRecordAtom *semtypes.MappingAtomicType
	byteStreamTy   semtypes.SemType
	roRecordTy     semtypes.SemType
	roRecordAtom   *semtypes.MappingAtomicType

	once          sync.Once
	fileInfoTy    semtypes.SemType
	fileInfoAtom  *semtypes.MappingAtomicType
	fileInfoArrTy semtypes.SemType
	fileInfoArrAt *semtypes.ListAtomicType
	watchEventTy  semtypes.SemType
	watchEvAtom   *semtypes.MappingAtomicType
}

func closedValueRecord(env semtypes.Env, valueTy semtypes.SemType) semtypes.SemType {
	md := semtypes.NewMappingDefinition()
	return md.Define(env, []semtypes.Field{semtypes.FieldFrom("value", valueTy, false, false)}, semtypes.Never)
}

func newFtpTypes(rt *runtime.Runtime) *ftpTypes {
	env := rt.GetTypeEnv()
	tc := semtypes.ContextFrom(env)
	t := &ftpTypes{env: env, jsonTy: semtypes.CreateJSON(tc)}
	jld := semtypes.NewListDefinition()
	t.jsonListTy = jld.Define(env, nil, semtypes.ListRest(t.jsonTy))
	t.jsonListAtom = semtypes.ToListAtomicType(env, t.jsonListTy)
	jmd := semtypes.NewMappingDefinition()
	t.jsonMapTy = jmd.Define(env, nil, t.jsonTy)
	t.jsonMapAtom = semtypes.ToMappingAtomicType(tc, t.jsonMapTy)
	bld := semtypes.NewListDefinition()
	t.byteArrTy = bld.Define(env, nil, semtypes.ListRest(semtypes.Byte))
	t.byteArrAtom = semtypes.ToListAtomicType(env, t.byteArrTy)
	rbld := semtypes.NewListDefinition()
	t.roByteArrTy = rbld.Define(env, nil, semtypes.ListRest(semtypes.Byte),
		semtypes.ListMutability(semtypes.CellMutabilityNone))
	t.roByteArrAtom = semtypes.ToListAtomicType(env, t.roByteArrTy)
	sld := semtypes.NewListDefinition()
	t.strArrTy = sld.Define(env, nil, semtypes.ListRest(semtypes.String))
	t.strArrAtom = semtypes.ToListAtomicType(env, t.strArrTy)
	ssld := semtypes.NewListDefinition()
	t.strArrArrTy = ssld.Define(env, nil, semtypes.ListRest(t.strArrTy))
	t.strArrArrAtom = semtypes.ToListAtomicType(env, t.strArrArrTy)
	t.byteRecordTy = closedValueRecord(env, t.byteArrTy)
	t.byteRecordAtom = semtypes.ToMappingAtomicType(tc, t.byteRecordTy)
	t.roRecordTy = closedValueRecord(env, t.roByteArrTy)
	t.roRecordAtom = semtypes.ToMappingAtomicType(tc, t.roRecordTy)
	sd := semtypes.NewStreamDefinition()
	t.byteStreamTy = sd.Define(env, t.byteArrTy, semtypes.Union(semtypes.Error, semtypes.Nil))
	return t
}

func bytesToItems(data []byte) []values.BalValue {
	items := make([]values.BalValue, len(data))
	for i, b := range data {
		items[i] = int64(b)
	}
	return items
}

func (t *ftpTypes) bytes(data []byte) *values.List {
	return values.NewList(t.byteArrTy, t.byteArrAtom, false, nil, 0, bytesToItems(data))
}

func (t *ftpTypes) readonlyBytes(data []byte) *values.List {
	return values.NewList(t.roByteArrTy, t.roByteArrAtom, true, nil, 0, bytesToItems(data))
}

func (t *ftpTypes) strings(items []string) *values.List {
	vals := make([]values.BalValue, len(items))
	for i, s := range items {
		vals[i] = s
	}
	return values.NewList(t.strArrTy, t.strArrAtom, false, nil, 0, vals)
}

// resolveRecordTypes loads FileInfo and WatchEvent from the module's typedesc helpers on first use.
func (t *ftpTypes) resolveRecordTypes(ctx *extern.Context) {
	t.once.Do(func() {
		t.fileInfoTy = typeFromHelper(ctx, "fileInfoType")
		t.fileInfoAtom = semtypes.ToMappingAtomicType(ctx.TypeCtx(), t.fileInfoTy)
		ld := semtypes.NewListDefinition()
		t.fileInfoArrTy = ld.Define(t.env, nil, semtypes.ListRest(t.fileInfoTy))
		t.fileInfoArrAt = semtypes.ToListAtomicType(t.env, t.fileInfoArrTy)
		t.watchEventTy = typeFromHelper(ctx, "watchEventType")
		t.watchEvAtom = semtypes.ToMappingAtomicType(ctx.TypeCtx(), t.watchEventTy)
	})
}

func typeFromHelper(ctx *extern.Context, name string) semtypes.SemType {
	handle, ok := ctx.LookupFunction(orgName, moduleName, name)
	if !ok {
		return semtypes.Never
	}
	td, err := ctx.InvokeFunction(handle, nil)
	if err != nil {
		return semtypes.Never
	}
	return td.(*values.TypeDesc).Constraint()
}

// fileInfoFields describes a file the way the VFS-backed FileInfo does.
type fileInfoFields struct {
	path     string
	uri      string
	rootURI  string
	friendly string
	logical  string
	scheme   string
	entry    remoteEntry
}

func extension(name string) string {
	i := strings.LastIndexByte(name, '.')
	if i <= 0 || i == len(name)-1 {
		return ""
	}
	return name[i+1:]
}

// depth follows VFS AbstractFileName.getDepth: the root is 0, otherwise segments + 1.
func depth(p string) int64 {
	if p == "/" || p == "" {
		return 0
	}
	return int64(strings.Count(p, "/") + 1)
}

func (t *ftpTypes) fileInfo(tc semtypes.Context, f fileInfoFields) *values.Map {
	fileType := "file"
	if f.entry.isDir {
		fileType = "folder"
	}
	size := f.entry.size
	if f.entry.isDir {
		size = 0
	}
	entries := []values.MapEntry{
		{Key: "path", Value: f.path},
		{Key: "size", Value: size},
		{Key: "lastModifiedTimestamp", Value: f.entry.modTime.UnixMilli()},
		{Key: "name", Value: path.Base(f.logical)},
		{Key: "isFolder", Value: f.entry.isDir},
		{Key: "isFile", Value: !f.entry.isDir},
		{Key: "pathDecoded", Value: f.logical},
		{Key: "extension", Value: extension(path.Base(f.logical))},
		{Key: "publicURIString", Value: f.friendly},
		{Key: "fileType", Value: fileType},
		{Key: "isAttached", Value: true},
		{Key: "isContentOpen", Value: false},
		{Key: "isExecutable", Value: f.entry.executable},
		{Key: "isHidden", Value: false},
		{Key: "isReadable", Value: f.entry.readable},
		{Key: "isWritable", Value: f.entry.writable},
		{Key: "depth", Value: depth(f.logical)},
		{Key: "scheme", Value: f.scheme},
		{Key: "uri", Value: f.uri},
		{Key: "rootURI", Value: f.rootURI},
		{Key: "friendlyURI", Value: f.friendly},
	}
	return values.NewMap(t.fileInfoTy, t.fileInfoAtom, false, entries)
}

func (t *ftpTypes) fileInfoList(items []values.BalValue) *values.List {
	return values.NewList(t.fileInfoArrTy, t.fileInfoArrAt, false, nil, 0, items)
}

// clientFileInfo is the FileInfo client->list returns: path is the server path and uri holds
// the full URI with the password, as VFS FileName.getURI does.
func clientFileInfo(t *ftpTypes, tc semtypes.Context, ep *endpoint, logical string, entry remoteEntry) *values.Map {
	return t.fileInfo(tc, fileInfoFields{
		path:     logical,
		uri:      vfsURI(ep, logical, false),
		rootURI:  vfsRootURI(ep, false),
		friendly: vfsURI(ep, logical, true),
		logical:  logical,
		scheme:   ep.protocol,
		entry:    entry,
	})
}

// listenerFileInfo is the FileInfo a listener passes to handlers: path is the full URI and
// uri the scheme-less URL path, both with the password, as in FtpListener.
func listenerFileInfo(t *ftpTypes, tc semtypes.Context, ep *endpoint, logical string, entry remoteEntry) *values.Map {
	full := vfsURI(ep, logical, false)
	return t.fileInfo(tc, fileInfoFields{
		path:     full,
		uri:      strings.TrimPrefix(full, ep.protocol+":"),
		rootURI:  vfsRootURI(ep, false),
		friendly: vfsURI(ep, logical, true),
		logical:  logical,
		scheme:   ep.protocol,
		entry:    entry,
	})
}

// mappingAtom returns the record shape of a mapping type. An inferred typedesc arrives as a union
// of equivalent intersections, so the first alternative stands for all of them.
func mappingAtom(tc semtypes.Context, t semtypes.SemType) *semtypes.MappingAtomicType {
	if atomic := semtypes.ToMappingAtomicType(tc, t); atomic != nil {
		return atomic
	}
	for _, alt := range semtypes.MappingAlternatives(tc, t) {
		if atomic := alt.Atomic(); atomic != nil {
			return atomic
		}
	}
	return nil
}

// relaxForBinding applies laxDataBinding: a null for an optional field is dropped, an absent
// nilable field becomes (), and members a closed record cannot hold are projected away.
func relaxForBinding(tc semtypes.Context, types *ftpTypes, value values.BalValue, target semtypes.SemType) values.BalValue {
	switch v := value.(type) {
	case *values.Map:
		atomic := mappingAtom(tc, semtypes.Intersect(target, semtypes.Mapping))
		if atomic == nil {
			return v
		}
		out := values.NewMap(types.jsonMapTy, types.jsonMapAtom, false, nil)
		for _, key := range v.Keys() {
			fieldTy := atomic.FieldInnerVal(key)
			member, _ := v.Get(key)
			if semtypes.IsNever(fieldTy) {
				continue
			}
			if member == nil && atomic.IsOptional(tc, key) && !semtypes.IsSubtype(tc, semtypes.Nil, fieldTy) {
				continue
			}
			out.Put(tc, key, relaxForBinding(tc, types, member, fieldTy))
		}
		for _, name := range atomic.FieldNames() {
			if _, present := v.Get(name); !present && !atomic.IsOptional(tc, name) &&
				semtypes.IsSubtype(tc, semtypes.Nil, atomic.FieldInnerVal(name)) {
				out.Put(tc, name, nil)
			}
		}
		return out
	case *values.List:
		listTy := semtypes.Intersect(target, semtypes.List)
		items := make([]values.BalValue, v.Len())
		for i := range items {
			memberTy := semtypes.ListMemberTypeInnerVal(tc, listTy, semtypes.IntConst(int64(i)))
			items[i] = relaxForBinding(tc, types, v.Get(i), memberTy)
		}
		return values.NewList(types.jsonListTy, types.jsonListAtom, false, nil, 0, items)
	}
	return value
}

type ftpModule struct {
	rt    *runtime.Runtime
	types *ftpTypes
}

func initFtpModule(rt *runtime.Runtime) {
	m := &ftpModule{rt: rt, types: newFtpTypes(rt)}
	m.registerClientExterns()
	m.registerCallerExterns()
	m.registerListenerExterns()
}

func init() {
	runtime.RegisterModuleInitializer(initFtpModule)
}
