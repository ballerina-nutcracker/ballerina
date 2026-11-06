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

// Only a wait-future-expr that is a variable-reference-expr naming a worker has
// the worker's return type as its eventual type (spec 7.7). Any other
// expression of type future<int>, even one that evaluates to the worker's
// future, has eventual type int|error.
function braced() returns int {
    worker w returns int {
        return 1;
    }
    int r = wait (w); // @error a braced worker name has eventual type int|error
    return r;
}

function conditional(boolean flag) returns int {
    worker a returns int {
        return 1;
    }
    worker b returns int {
        return 2;
    }
    int r = wait (flag ? a : b); // @error a conditional over workers has eventual type int|error
    return r;
}

public function main() {
    io:println(braced());
    io:println(conditional(true));
}
