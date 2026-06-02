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
    checkpanic run();
}

function run() returns error? {
    // json array to json (including null elements).
    json arr = [1, (), 3];
    json arrOut = check arr.fromJsonWithType(json);
    io:println(arrOut); // @output [1,null,3]

    // json map to json.
    json m = {"a": 1, "b": ()};
    json mapOut = check m.fromJsonWithType(json);
    io:println(mapOut); // @output {"a":1,"b":null}

    // nested json to json.
    json nested = [[1, 2], [3, 4]];
    json nestedOut = check nested.fromJsonWithType(json);
    io:println(nestedOut); // @output [[1,2],[3,4]]

    json rec = {"items": [1, 2], "meta": {"n": 2}};
    json recOut = check rec.fromJsonWithType(json);
    io:println(recOut); // @output {"items":[1,2],"meta":{"n":2}}
    return;
}
