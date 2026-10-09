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

function ()? saved = ();

function run(function ()? f) returns int {
    if f is function () {
        f();
    }
    return 0;
}

// A closure built for one element runs between the test and the use for the
// next one, so narrowing x there would let a string reach an int.
public function main() {
    int|string x = 1;
    int[] xs = [1, 2];
    int[] result = from int i in xs
        let int a = x is int ? run(saved) + x : 0 // @error saved may have assigned a string to x
        let function () setter = function () {
            x = "changed";
        }
        let int b = save(setter)
        select a + b + i;
    io:println(result);
}

function save(function () f) returns int {
    saved = f;
    return 0;
}
