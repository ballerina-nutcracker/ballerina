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

type FrozenField record {|
    readonly int id;
|};

function update(map<int> value) {
    value["id"] = 99;
}

function isReadonly(any value) returns boolean {
    return value is readonly;
}

public function main() {
    string key = "id";
    map<int> explicitField = {readonly id: 1, [key]: 2};
    io:println(explicitField["id"]); // @output 2
    io:println((trap update(explicitField)) is error); // @output true

    FrozenField inherentField = {[key]: 4, id: 3};
    io:println(inherentField.id); // @output 4
    io:println((trap update(inherentField)) is error); // @output true

    readonly & map<int> wholeMap = {id: 5, [key]: 6};
    io:println(wholeMap["id"]); // @output 6
    io:println(isReadonly(wholeMap)); // @output true
    io:println((trap update(wholeMap)) is error); // @output true
}
