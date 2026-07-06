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
type F1 function(int) returns string;
type F2 function(string) returns int;

function foo(int a) returns string {
        int _ = a;
    return "foo";
    }

function bar(string a) returns int {
        string _ = a;
    return 5;
    }

public function main() {
    F1 f1 = foo;
    F2 f2 = bar;
    boolean test = f1 == f2; // @error
    io:println(test);
}
