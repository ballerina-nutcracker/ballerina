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

function run(function ()? f) {
    if f is function () {
        f();
    }
}

// A closure saved on one iteration runs between the test and the use on the
// next one, so narrowing x there would let a string reach an int.
function whileLoop() {
    int|string x = 1;
    function ()? saved = ();
    int i = 0;
    while i < 2 {
        if x is int {
            run(saved);
            int y = x; // @error saved may have assigned a string to x
            io:println(y);
        }
        saved = function () {
            x = "str";
        };
        i += 1;
    }
}

function foreachLoop() {
    int|string x = 1;
    function ()? saved = ();
    foreach int i in 0 ..< 2 {
        if x is int {
            run(saved);
            int y = x; // @error saved may have assigned a string to x
            io:println(y + i);
        }
        saved = function () {
            x = "str";
        };
    }
}

public function main() {
    whileLoop();
    foreachLoop();
}
