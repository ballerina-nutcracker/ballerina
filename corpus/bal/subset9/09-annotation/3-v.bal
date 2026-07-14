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

type Meta readonly & record {|
    string name;
    int code;
|};

annotation Meta fnInfo on function;
annotation paramInfo on parameter;
const annotation Meta constInfo on source const;
const annotation Meta varInfo on source var;

@constInfo {name: "limit", code: 1}
const LIMIT = 10;

@varInfo {name: "offset", code: 2}
int offset = 1;

@fnInfo {name: "add", code: 3}
function add(@paramInfo int left, @paramInfo int right) returns int {
    return left + right;
}

public function main() {
    io:println(add(2, 3)); // @output 5
    io:println(LIMIT + offset); // @output 11
}
