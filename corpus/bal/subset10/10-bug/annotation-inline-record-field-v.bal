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

type Info record {|
    int value;
|};

annotation Info recordFieldInfo on record field;

function paramValue(record {|
    @recordFieldInfo {value: 1}
    int b;
|} param) returns int {
    return param.b;
}

record {|
    @recordFieldInfo {value: 2}
    int c;
|} moduleVar = {c: 2};

function returnValue() returns record {|
    @recordFieldInfo {value: 3}
    int d;
|} {
    return {d: 3};
}

public function main() {
    io:println(paramValue({b: 1})); // @output 1
    io:println(moduleVar.c); // @output 2
    io:println(returnValue().d); // @output 3
}
