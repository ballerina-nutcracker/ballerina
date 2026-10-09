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

// Calling f without an argument runs the default of its function type, which
// assigns x through a closure. So x cannot stay narrowed after the descriptor.
// jBallerina accepts this narrowing: https://github.com/ballerina-platform/ballerina-lang/issues/44765
public function main() {
    int|string x = 1;
    if x is int {
        function (int a = run(function () returns int {
                    x = "s";
                    return 1;
                })) returns int f = id;
        int _ = f();
        int y = x; // @error the default assigns x, so it is no longer narrowed to int
        io:println(y);
    }
}

function run(function () returns int g) returns int {
    return g();
}

function id(int a) returns int {
    return a;
}
