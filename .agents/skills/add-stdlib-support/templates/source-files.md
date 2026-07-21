# Source-file templates for a `ballerina/<name>` stdlib package

Every `.bal` and `.go` file needs the Apache 2.0 license header shown below.

## `.bal` template (with the required license header)

```ballerina
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

# Doc comment for the public function.
# + arg - Description
# + return - Description
public isolated function publicFn(string arg) returns string|error {
    return externFn(arg);
}

isolated function externFn(string arg) returns string|error = external;
```

## `native/<name>.go` template

```go
// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
// [...same Apache 2.0 header as above, in Go comment form...]

package native

import (
    "ballerina/runtime"
    "ballerina/runtime/extern"
    "ballerina/values"
)

const (
    orgName    = "ballerina"
    moduleName = "<name>"
)

func externFnExtern(rt *runtime.Runtime) extern.NativeFunc {
    return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
        // implementation
        return nil, nil
    }
}

func init<Name>Module(rt *runtime.Runtime) {
    runtime.RegisterExternFunction(rt, orgName, moduleName, "externFn", externFnExtern(rt))
}

func init() {
    runtime.RegisterModuleInitializer(init<Name>Module)
}
```
