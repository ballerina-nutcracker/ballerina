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

type PetByAge record {
    int age;
    string nickname?;
};

type PetByType record {
    "Cat"|"Dog" pet_type;
    boolean hunts?;
};

type Pet PetByAge|PetByType;

type MapUnion map<int>|map<string>;
type IntStrArr int[]|string[];
type IntFloat int|float;
type ByteOrString byte|string;

public function main() {
    checkpanic run();
}

function run() returns error? {
    json petJson = {"nickname": "Fido", "pet_type": "Dog", "age": 4};
    Pet pet = check petJson.fromJsonWithType(Pet);
    io:println(pet); // @output {"nickname":"Fido","pet_type":"Dog","age":4}

    json mapJson = {"k": 2};
    MapUnion mapVal = check mapJson.fromJsonWithType(MapUnion);
    io:println(mapVal); // @output {"k":2}

    json arrJson = [1, 2];
    IntStrArr arrVal = check arrJson.fromJsonWithType(IntStrArr);
    io:println(arrVal); // @output [1,2]

    json floatJson = 12.0;
    IntFloat numVal = check floatJson.fromJsonWithType(IntFloat);
    io:println(numVal is float); // @output true

    json floatAsByte = 200.0;
    ByteOrString asByte = check floatAsByte.fromJsonWithType(ByteOrString);
    io:println(asByte); // @output 200
    return;
}
