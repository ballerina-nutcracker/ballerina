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

package array

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.array"
)

func initArrayModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "push", func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		if list, ok := args[0].(*values.List); ok {
			list.Append(ctx.TypeCtx, args[1:]...)
			return nil, nil
		}
		return nil, fmt.Errorf("first argument must be an array")
	})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "length", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		if list, ok := args[0].(*values.List); ok {
			return int64(list.Len()), nil
		}
		return nil, fmt.Errorf("first argument must be an array")
	})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "indexOf", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		list, ok := args[0].(*values.List)
		if !ok {
			return nil, fmt.Errorf("first argument must be an array")
		}
		val := args[1]
		startIndex := int64(0)
		if len(args) > 2 && args[2] != nil {
			if si, ok := args[2].(int64); ok {
				startIndex = si
			}
		}
		for i := int(startIndex); i < list.Len(); i++ {
			if values.DeepEquals(list.Get(i), val) {
				return int64(i), nil
			}
		}
		return nil, nil
	})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "remove", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		list, ok := args[0].(*values.List)
		if !ok {
			return nil, fmt.Errorf("first argument must be an array")
		}
		index, ok := args[1].(int64)
		if !ok {
			return nil, fmt.Errorf("second argument must be an int")
		}
		return list.RemoveAt(int(index)), nil
	})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "removeAll", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		list, ok := args[0].(*values.List)
		if !ok {
			return nil, fmt.Errorf("first argument must be an array")
		}
		list.Clear()
		return nil, nil
	})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toBase16", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		list, ok := args[0].(*values.List)
		if !ok {
			return nil, fmt.Errorf("first argument must be a byte array")
		}
		return hex.EncodeToString(listToBytes(list)), nil
	})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toBase64", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		list, ok := args[0].(*values.List)
		if !ok {
			return nil, fmt.Errorf("first argument must be a byte array")
		}
		return base64.StdEncoding.EncodeToString(listToBytes(list)), nil
	})
}

func listToBytes(list *values.List) []byte {
	b := make([]byte, list.Len())
	for i := range list.Len() {
		b[i] = byte(list.Get(i).(int64))
	}
	return b
}

func init() {
	runtime.RegisterModuleInitializer(initArrayModule)
}
