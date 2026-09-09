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

// The opaque marker is honoured only in a ballerina/lang.* package. Here it is
// just a documentation line, so add is an ordinary function: it keeps its own
// signature, its default and its named arguments.

# Adds two numbers.
# @opaque
#
# + a - the first number
# + b - the second number
# + return - the sum
function add(int a, int b = 2) returns int {
    return a + b;
}

# @opaque
public function main() {
    io:println(add(1)); // @output 3
    io:println(add(1, 5)); // @output 6
    io:println(add(a = 1, b = 5)); // @output 6
    io:println(add(1, b = 10)); // @output 11
}
