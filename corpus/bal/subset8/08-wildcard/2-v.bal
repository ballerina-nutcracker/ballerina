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

// @labels local-init-var-decl-stmt function-call-expr if-else-stmt
import ballerina/io;

function stringFoo(string s) {
    io:println("This is " + s);
}

function intFoo(int i) {
    int _ = i;
    io:println("This is integer");
}

function foo(int|string x) {
    if x is int {
        intFoo(x);
    }
    else {
        string _ = x;
        stringFoo(x);
    }
}

public function main() {
    foo(2); // @output This is integer
    foo("test str"); // @output This is test str
}
