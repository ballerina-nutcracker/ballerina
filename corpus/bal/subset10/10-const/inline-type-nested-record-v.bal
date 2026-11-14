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

const record {| int y = 2; |}[] L = [{}];
const map<record {| int y = 2; |}> M = {a: {}};
const [record {| int y = 2; |}, record {| int y = 3; |}...] T = [{}, {}];
const record {| int y = 2; |}|int U = {};
const record {| record {| record {| int z = 3; |} q; |}[] r; |} D = {r: [{q: {}}]};

function isReadonly(any v) returns boolean => v is readonly;

public function main() {
    io:println(L); // @output [{"y":2}]
    io:println(M); // @output {"a":{"y":2}}
    io:println(T); // @output [{"y":2},{"y":3}]
    io:println(U); // @output {"y":2}
    io:println(D); // @output {"r":[{"q":{"z":3}}]}
    io:println(isReadonly(D.r[0].q)); // @output true
}
