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

import crossmodulerecorddefaultsexportedconst.base;
import ballerina/io;

type Default base:DEFAULT;

function isDefault(any v) returns boolean {
    return v is Default;
}

public function main() {
    io:println(base:DEFAULT); // @output {"x":5,"s":"a","leaf":{"n":1}}
    5 x = base:DEFAULT.x;
    io:println(x); // @output 5
    io:println(isDefault(base:DEFAULT)); // @output true
    io:println(base:DEFAULT.leaf is readonly); // @output true
    base:Config c = {};
    io:println(c == base:DEFAULT); // @output true
}
