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
// @labels local-init-var-decl-stmt compound-assignment-stmt function-defn implicit-anon-function-expr query-expr
import ballerina/io;

function f(int '_) returns int => '_ * 10;

public function main() {
    int _ = 5;
    int '_ = 1;
    '_ += 2;
    io:println('_); // @output 3
    io:println(f(4)); // @output 40
    arrow();
    queries();
}

function arrow() {
    function (int) returns int g = '_ => '_ + 1;
    io:println(g(6)); // @output 7
}

function queries() {
    io:println(from int '_ in [1, 2, 3] let int k = 1 group by k select [k, ['_, 0].length()]); // @output [[1,4]]
    io:println(from int '_ in [1, 2, 3] select '_ * 2); // @output [2,4,6]
}
