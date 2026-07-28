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
    string name;
    int code;
|};

annotation Info info on type;
annotation marker on type;

@info {name: "person", code: 23}
@marker
type Person record {|
    string name;
|};

function personType() returns typedesc<Person> {
    return Person;
}

public function main() {
    typedesc<Person> descriptor = personType();
    typedesc<Person> copied = descriptor;

    Info? infoValue = copied.@info;
    if infoValue is Info {
        io:println(infoValue.name); // @output person
        io:println(infoValue.code); // @output 23
    }

    boolean? markerValue = copied.@marker;
    if markerValue is boolean {
        io:println(markerValue); // @output true
    }
}
