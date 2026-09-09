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

// An omitted default is supplied wherever an opaque invocation can occur:
// module-level variable initializers, ordinary function bodies, lambdas and
// nested expressions.
int[] letters = [10, 20, 30, 20];
int? moduleLevel = letters.indexOf(20);

function inFunctionBody() returns int? {
    int[] arr = [1, 2, 3, 2];
    return arr.indexOf(2);
}

public function main() {
    io:println(moduleLevel); // @output 1
    io:println(inFunctionBody()); // @output 1

    var lambda = function() returns int? {
        int[] arr = [5, 6, 7, 6];
        return arr.indexOf(6);
    };
    io:println(lambda()); // @output 1

    int[] nested = [1, 2, 3];
    io:println(nested.indexOf(nested.indexOf(2) ?: 0)); // @output 0
}
