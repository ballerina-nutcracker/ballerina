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

class P {
    int base = 3;

    function m() returns int {
        var f = function(int a, int b = self.base) returns int => a + b;
        return f(1);
    }
}

public function main() {
    int base = 100;
    var add = function(int x, int y = base) returns int => x + y;
    var callAdd = function() returns int => add(1);
    io:println(callAdd()); // @output 101

    function (int x, int y = base) returns int typed = function(int x, int y) returns int => x + y;
    var callTyped = function() returns int => typed(2);
    io:println(callTyped()); // @output 102

    P p = new;
    io:println(p.m()); // @output 4
}
