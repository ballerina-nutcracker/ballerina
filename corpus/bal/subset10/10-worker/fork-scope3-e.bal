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

// A function-level worker's name is not in scope in the initialization
// statements, including the bodies of a fork's workers there (spec 7.3.2).
function forkInInitWaitsOnFunctionWorker() returns int {
    fork {
        worker f returns int {
            return wait w; // @error w is not in scope in the initialization statements
        }
    }
    worker w returns int {
        return 2;
    }
    return wait f;
}

public function main() {
    io:println(forkInInitWaitsOnFunctionWorker());
}
