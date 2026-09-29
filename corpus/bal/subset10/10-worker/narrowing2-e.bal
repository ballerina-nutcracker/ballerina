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

// A worker name is not a variable: it stands for a single-use future the
// declaring function publishes, so there is no second read a narrowed type
// could apply to.
function typeTest() {
    worker w returns int {
        return 1;
    }
    if w is future<int> { // @error cannot narrow a worker reference
        int result = wait w;
        io:println(result);
    }
}

function matchStatement() {
    worker w returns int {
        return 1;
    }
    match w { // @error cannot narrow a worker reference
        _ => {
            io:println("matched");
        }
    }
}

public function main() {
    typeTest();
    matchStatement();
}
