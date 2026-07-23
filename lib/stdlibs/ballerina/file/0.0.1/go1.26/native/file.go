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
	"os"
	"path/filepath"
	"sync"
	"time"

	"ballerina-lang-go/decimal"
	"ballerina-lang-go/platform/pal"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "file"
)

func fileError(typeName, msg string) values.BalValue {
	return values.NewError(semtypes.ERROR, msg, nil, typeName, nil)
}

func goTimeToUtc(ctx *extern.Context, t time.Time) *values.List {
	t = t.UTC()
	epochSec := t.Unix()
	nanos := decimal.FromInt64(int64(t.Nanosecond()))
	nanosPerSec := decimal.FromInt64(1_000_000_000)
	frac, _ := nanos.Quo(nanosPerSec)
	bld := semtypes.NewListDefinition()
	utcTy := bld.TupleTypeWrappedRo(ctx.Env.TypeEnv, semtypes.INT, semtypes.DECIMAL)
	atomic := semtypes.ToListAtomicType(ctx.TypeCtx, utcTy)
	return values.NewList(utcTy, atomic, true, nil, 2, []values.BalValue{epochSec, frac})
}

func buildMetaData(ctx *extern.Context, info *pal.FileInfo) *values.Map {
	mmd := semtypes.NewMappingDefinition()
	ty := mmd.DefineMappingTypeWrapped(ctx.Env.TypeEnv, nil, semtypes.STRING)
	return values.NewMap(ty, semtypes.ToMappingAtomicType(ctx.TypeCtx, ty), false, []values.MapEntry{
		{Key: "absPath", Value: info.AbsPath},
		{Key: "size", Value: info.Size},
		{Key: "modifiedTime", Value: goTimeToUtc(ctx, info.ModifiedAt)},
		{Key: "dir", Value: info.IsDir},
		{Key: "readable", Value: info.IsReadable},
		{Key: "writable", Value: info.IsWritable},
	})
}

func absPath(rt *runtime.Runtime, path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	cwd, err := rt.Platform().FS.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(cwd, path)), nil
}

func initFileModule(rt *runtime.Runtime) {
	var (
		once      sync.Once
		metaArrTy semtypes.SemType
	)
	ensureTypes := func() {
		once.Do(func() {
			env := rt.GetTypeEnv()
			bld := semtypes.NewListDefinition()
			metaArrTy = bld.DefineListTypeWrappedWithEnvSemType(env, semtypes.MAPPING)
		})
	}

	runtime.RegisterExternFunction(rt, orgName, moduleName, "getCurrentDir",
		func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
			cwd, err := rt.Platform().FS.Getwd()
			if err != nil {
				return "", nil
			}
			return cwd, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "createDir",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			dir, _ := args[0].(string)
			option, _ := args[1].(string)
			var err error
			if option == "RECURSIVE" {
				err = rt.Platform().FS.MkdirAll(dir)
			} else {
				err = rt.Platform().FS.Mkdir(dir)
			}
			if err == nil {
				return nil, nil
			}
			if os.IsExist(err) {
				return fileError("InvalidOperationError", "File already exists. Failed to create the file: "+dir), nil
			}
			if os.IsPermission(err) {
				return fileError("PermissionError", "Permission denied. Failed to create the file: "+dir), nil
			}
			return fileError("FileSystemError", "IO error while creating the file "+dir), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "remove",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			path, _ := args[0].(string)
			option, _ := args[1].(string)
			cwd, cwdErr := rt.Platform().FS.Getwd()
			if cwdErr != nil {
				return fileError("FileSystemError", "Error while deleting the file/directory: "+cwdErr.Error()), nil
			}
			absTarget, _ := absPath(rt, path)
			if absTarget == filepath.Clean(cwd) {
				return fileError("InvalidOperationError", "Cannot delete the current working directory "+cwd), nil
			}
			_, statErr := rt.Platform().FS.Stat(path)
			if statErr != nil {
				if os.IsNotExist(statErr) {
					return fileError("FileNotFoundError", "File not found: "+absTarget), nil
				}
				return fileError("FileSystemError", "Error while deleting the file/directory: "+statErr.Error()), nil
			}
			var err error
			if option == "RECURSIVE" {
				err = rt.Platform().FS.RemoveAll(path)
			} else {
				err = rt.Platform().FS.Remove(path)
			}
			if err == nil {
				return nil, nil
			}
			if os.IsPermission(err) {
				return fileError("PermissionError", "Error while deleting the file/directory: "+err.Error()), nil
			}
			return fileError("FileSystemError", "Error while deleting the file/directory: "+err.Error()), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "rename",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			oldPath, _ := args[0].(string)
			newPath, _ := args[1].(string)
			_, statErr := rt.Platform().FS.Stat(oldPath)
			if statErr != nil {
				if os.IsNotExist(statErr) {
					absOld, _ := absPath(rt, oldPath)
					return fileError("FileNotFoundError", "File not found: "+absOld), nil
				}
			}
			err := rt.Platform().FS.Rename(oldPath, newPath)
			if err == nil {
				return nil, nil
			}
			if os.IsExist(err) {
				return fileError("InvalidOperationError", "File already exists in the new path "+newPath), nil
			}
			if os.IsPermission(err) {
				return fileError("PermissionError", err.Error()), nil
			}
			return fileError("FileSystemError", err.Error()), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "create",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			path, _ := args[0].(string)
			err := rt.Platform().FS.CreateFile(path)
			if err == nil {
				return nil, nil
			}
			if os.IsExist(err) {
				return fileError("InvalidOperationError", "File already exists. Failed to create the file: "+path), nil
			}
			if os.IsPermission(err) {
				return fileError("PermissionError", "Permission denied. Failed to create the file: "+path), nil
			}
			if os.IsNotExist(err) {
				return fileError("FileSystemError", "The file does not exist in path "+path), nil
			}
			return fileError("FileSystemError", "IO error occurred while creating the file "+path), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "getRawMetaData",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			path, _ := args[0].(string)
			info, err := rt.Platform().FS.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					return fileError("FileNotFoundError", "File not found: "+path), nil
				}
				return fileError("FileSystemError", err.Error()), nil
			}
			return buildMetaData(ctx, info), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "readDirRaw",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			ensureTypes()
			path, _ := args[0].(string)
			info, err := rt.Platform().FS.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					return fileError("FileNotFoundError", "File not found: "+path), nil
				}
				return fileError("FileSystemError", err.Error()), nil
			}
			if !info.IsDir {
				return fileError("InvalidOperationError", "File in path "+path+" is not a directory"), nil
			}
			entries, err := rt.Platform().FS.ReadDir(path)
			if err != nil {
				if os.IsPermission(err) {
					return fileError("PermissionError", err.Error()), nil
				}
				return fileError("FileSystemError", err.Error()), nil
			}
			items := make([]values.BalValue, len(entries))
			for i, entry := range entries {
				e := entry
				items[i] = buildMetaData(ctx, &e)
			}
			atomic := semtypes.ToListAtomicType(ctx.TypeCtx, metaArrTy)
			return values.NewList(metaArrTy, atomic, false, nil, 0, items), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "copy",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			src, _ := args[0].(string)
			dst, _ := args[1].(string)
			opts := pal.CopyOptions{}
			for _, arg := range args[2:] {
				switch arg {
				case "REPLACE_EXISTING":
					opts.ReplaceExisting = true
				case "COPY_ATTRIBUTES":
					opts.CopyAttributes = true
				case "NO_FOLLOW_LINKS":
					opts.NoFollowLinks = true
				}
			}
			_, statErr := rt.Platform().FS.Stat(src)
			if statErr != nil && os.IsNotExist(statErr) {
				return fileError("FileNotFoundError", "File not found: "+src), nil
			}
			err := rt.Platform().FS.Copy(src, dst, opts)
			if err == nil {
				return nil, nil
			}
			if os.IsNotExist(err) {
				return fileError("FileNotFoundError", "The target directory does not exist: "+err.Error()), nil
			}
			return fileError("FileSystemError", "An error occurred when copying the file/s: "+err.Error()), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "createTemp",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			suffix := ""
			prefix := ""
			dir := ""
			if s, ok := args[0].(string); ok {
				suffix = s
			}
			if p, ok := args[1].(string); ok {
				prefix = p
			}
			if d, ok := args[2].(string); ok {
				dir = d
			}
			path, err := rt.Platform().FS.CreateTemp(prefix, suffix, dir)
			if err != nil {
				return fileError("FileSystemError", "Error occurred while creating temporary file. "+err.Error()), nil
			}
			return path, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "createTempDir",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			suffix := ""
			prefix := ""
			dir := ""
			if s, ok := args[0].(string); ok {
				suffix = s
			}
			if p, ok := args[1].(string); ok {
				prefix = p
			}
			if d, ok := args[2].(string); ok {
				dir = d
			}
			path, err := rt.Platform().FS.CreateTempDir(prefix, suffix, dir)
			if err != nil {
				return fileError("FileSystemError", "Error occurred while creating temporary directory. "+err.Error()), nil
			}
			return path, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "test",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			path, _ := args[0].(string)
			option, _ := args[1].(string)
			switch option {
			case "EXISTS":
				_, err := rt.Platform().FS.Stat(path)
				return err == nil, nil
			case "IS_DIR":
				info, err := rt.Platform().FS.Stat(path)
				if err != nil {
					return false, nil
				}
				return info.IsDir, nil
			case "IS_SYMLINK":
				info, err := rt.Platform().FS.Lstat(path)
				if err != nil {
					return false, nil
				}
				return info.IsSymlink, nil
			case "READABLE":
				info, err := rt.Platform().FS.Stat(path)
				if err != nil {
					return false, nil
				}
				return info.IsReadable, nil
			case "WRITABLE":
				info, err := rt.Platform().FS.Stat(path)
				if err != nil {
					return false, nil
				}
				return info.IsWritable, nil
			default:
				return fileError("InvalidOperationError", "Unsupported test option."), nil
			}
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "charAt",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			input, _ := args[0].(string)
			index, _ := args[1].(int64)
			length := int64(len([]rune(input)))
			if index > length {
				return fileError("GenericError", fmt.Sprintf("Character index %d is greater then path string length %d", index, length)), nil
			}
			runes := []rune(input)
			return string(runes[index : index+1]), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "getAbsolutePath",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			path, _ := args[0].(string)
			abs, err := absPath(rt, path)
			if err != nil {
				return fileError("InvalidPathError", "Invalid path "+path), nil
			}
			return abs, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "resolve",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			path, _ := args[0].(string)
			target, err := rt.Platform().FS.Readlink(path)
			if err == nil {
				return target, nil
			}
			if os.IsNotExist(err) {
				return fileError("FileNotFoundError", "File does not exist at "+path), nil
			}
			if os.IsPermission(err) {
				return fileError("SecurityError", "Security error for "+path), nil
			}
			// On most systems, EINVAL means "not a symlink".
			return fileError("NotLinkError", "Path is not a symbolic link "+path), nil
		})
}

func init() {
	runtime.RegisterModuleInitializer(initFileModule)
}
