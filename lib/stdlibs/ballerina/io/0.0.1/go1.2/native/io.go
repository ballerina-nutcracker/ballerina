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

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "io"

	stdoutStream = int64(1)
	stderrStream = int64(2)
)

func Write(rt *runtime.Runtime, stream int64, newline bool, vals []values.BalValue) {
	parts := make([]string, len(vals))
	visited := make(map[uintptr]bool)
	for i, v := range vals {
		parts[i] = values.String(v, visited)
	}
	joined := strings.Join(parts, "")
	var out []byte
	if newline {
		out = fmt.Appendln(nil, joined)
	} else {
		out = []byte(joined)
	}
	pio := rt.Platform().IO
	if stream == stderrStream {
		_, _ = pio.Stderr(out)
	} else {
		_, _ = pio.Stdout(out)
	}
}

func externPrintExtern(rt *runtime.Runtime) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		stream, _ := args[0].(int64)
		newLine, _ := args[1].(bool)
		var vals []values.BalValue
		if list, ok := args[2].(*values.List); ok {
			for i := 0; i < list.Len(); i++ {
				vals = append(vals, list.Get(i))
			}
		}
		Write(rt, stream, newLine, vals)
		return nil, nil
	}
}

func initIOModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externPrint", externPrintExtern(rt))
}

func init() {
	runtime.RegisterModuleInitializer(initIOModule)
}
