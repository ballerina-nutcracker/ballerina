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

import ballerina/io;

public int a = 0;

function foo() {
    a += 1;
}

function readAndBump() returns int {
    lock {
        // JBalelrina don't detect this as an error but spec specifically says:
        // "A function or method can be called in the lock statement only if the type of the function is isolated."
        foo(); // @error
        return 1;
    }
}

public function main() {
    io:println(readAndBump());
}
