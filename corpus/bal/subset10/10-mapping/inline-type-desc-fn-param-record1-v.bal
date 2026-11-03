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

type A record {|
    function (record {| int a = 1; int b; |} r = {b: 2}) returns int f;
    int m;
|};

type B record {|
    function (record {| int a = 1; int b; |} r = {b: 2}) returns int f;
    string m;
|};

function sum(record {| int a = 1; int b; |} r = {b: 2}) returns int => r.a + r.b;

public function main() {
    any h = sum;
    A|B x = {f: <function (record {| int a = 1; int b; |} r = {b: 2}) returns int>h, m: 1};
    io:println(x is A); // @output true
    if x is A {
        function (record {| int a = 1; int b; |} r = {b: 2}) returns int g = x.f;
        io:println(g()); // @output 3
    }
}
