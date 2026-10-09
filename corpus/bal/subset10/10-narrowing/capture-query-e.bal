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

// Every clause after the initial collection repeats, so x, captured in a later
// clause, is never narrowed in any of them.
public function main() {
    int|string x = 1;
    int[] xs = [1, 2];
    int[] result = from int i in xs
        let int a = x is int ? x : 0 // @error a later capture in the query reaches this use
        let function () setter = function () {
            x = "changed";
        }
        let int b = run(setter)
        select a + b + i;
    io:println(result);
}

function run(function () f) returns int {
    f();
    return 0;
}
