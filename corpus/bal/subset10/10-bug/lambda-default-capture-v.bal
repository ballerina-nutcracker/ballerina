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

int base = 7;

public function main() {
    io:println(base); // @output 7
    int base = 100;
    var add = function(int x, int y = base) returns int => x + y;
    io:println(add(1)); // @output 101
    base = 200;
    io:println(add(1)); // @output 201
    io:println(add(1, 2)); // @output 3

    string greeting = "hi";
    var greet = function(string name, string g = greeting + " " + name) returns string => g;
    io:println(greet("a")); // @output hi a

    foreach int i in 0 ..< 2 {
        string s = "v" + i.toString();
        var f = function(int x, string y = s + x.toString()) returns string => y;
        io:println(f(7)); // @output v07
                          // @output v17
    }

    var apply = function(function (int x, int y = base) returns int h) returns int => h(1);
    io:println(apply(function(int x, int y) returns int => x + y)); // @output 201

    final int fieldBase = 300;
    var readField = function(record {| int a = fieldBase; |} r) returns int => r.a;
    io:println(readField({})); // @output 300
}
