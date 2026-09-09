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

// A final module variable or a parameter is immutable but has no
// compile-time value: a default reading it stays a runtime-only default.
final int SEED = 4;

type FromFinal record {|
    int x = SEED;
    int y = 1;
|};

const FromFinal A = {x: 2};

function build(int p) returns int {
    record {|int z = p; int w = 3;|} r = {};
    return r.z + r.w;
}

function sumInLoop() returns int {
    int total = 0;
    int i = 0;
    while i < 2 {
        record {|int z = 2; int w = 1;|} r = {};
        total += r.z + r.w;
        i += 1;
    }
    foreach int j in 0 ..< 2 {
        record {|int z = 10;|} r = {};
        total += r.z + j;
    }
    return total;
}

public function main() {
    io:println(sumInLoop()); // @output 27
    io:println(A); // @output {"x":2,"y":1}
    FromFinal r = {};
    io:println(r); // @output {"x":4,"y":1}
    io:println(build(5)); // @output 8
}
