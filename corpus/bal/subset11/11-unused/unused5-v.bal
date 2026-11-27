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

// @productions foreach-stmt from-clause join-clause
import ballerina/io;

public function main() {
    (int|error)[] values = [1, error("boom")];
    foreach var v in values { // @warn
        io:println("item"); // @output item
        // @output item
    }
    int[] r = from var w in values // @warn
              join var j in values on 1 equals 1 // @warn
              select 0;
    io:println(r); // @output [0,0,0,0]
}
