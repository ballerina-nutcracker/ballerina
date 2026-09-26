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

public function main() {
    io:println(describe(1)); // @output one
    io:println(describe(3)); // @output three
    io:println(describe(200)); // @output 200
    io:println(describe(0)); // @output other
}

function describe(int:Unsigned8 x) returns string {
    if x == 1 {
        return "one";
    }
    if x == 3 {
        return "three";
    }
    return x == 200 ? "200" : "other";
}
