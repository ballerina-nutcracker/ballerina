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

type Rec record {
    int id;
};

type Strings record {|
    int id;
    string...;
|};

string moduleKey = "a";
int calls = 0;

function nextKey() returns string {
    calls += 1;
    return "k" + calls.toString();
}

public function main() {
    string localKey = "b";
    map<int> m = {[moduleKey]: 1, [localKey]: 2, ["c" + "d"]: 3, e: 4};
    io:println(m); // @output {"a":1,"b":2,"cd":3,"e":4}

    var inferred = {[localKey]: 5, name: "x"};
    io:println(inferred); // @output {"b":5,"name":"x"}
    int|string? b = inferred[localKey];
    io:println(b); // @output 5

    Rec r = {id: 1, [localKey]: "rest"};
    io:println(r); // @output {"id":1,"b":"rest"}

    Strings s = {id: 2, [moduleKey]: "s"};
    io:println(s); // @output {"id":2,"a":"s"}

    map<int> ordered = {[nextKey()]: 1, [nextKey()]: 2};
    io:println(ordered); // @output {"k1":1,"k2":2}
}
