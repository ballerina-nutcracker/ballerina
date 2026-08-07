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

// Two open records sharing no required fields — union selection via isConvertibleMapping.
type Cat record {
    string name;
    int lives;
};

type Dog record {
    string name;
    string breed;
};

type Animal Cat|Dog;

// Map<string> vs map<int> — exercises isConvertibleMapping for each union member.
type StrMap map<string>;
type IntMap map<int>;
type StrOrInt StrMap|IntMap;

public function main() returns error? {
    json catJson = {"name": "Whiskers", "lives": 9};
    Animal cat = check catJson.fromJsonWithType(Animal);
    io:println(cat); // @output {"name":"Whiskers","lives":9}

    json dogJson = {"name": "Rex", "breed": "Labrador"};
    Animal dog = check dogJson.fromJsonWithType(Animal);
    io:println(dog); // @output {"name":"Rex","breed":"Labrador"}

    // Map union: string values → StrMap selected.
    json strMapJson = {"a": "hello", "b": "world"};
    StrOrInt strMap = check strMapJson.fromJsonWithType(StrOrInt);
    io:println(strMap); // @output {"a":"hello","b":"world"}

    // Map union: int values → IntMap selected.
    json intMapJson = {"x": 1, "y": 2};
    StrOrInt intMap = check intMapJson.fromJsonWithType(StrOrInt);
    io:println(intMap); // @output {"x":1,"y":2}

    // Neither member matches — must produce an error.
    json neither = {"name": "Whiskers"};
    io:println(neither.fromJsonWithType(Animal) is error); // @output true
    return;
}
