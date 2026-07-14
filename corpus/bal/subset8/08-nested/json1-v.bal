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

type J ()|boolean|int|float|string|J[]|map<J>;

public function main() {
    J j = [
        {name: "James", age: 100, children: ["Jack", "Jane"], married: true}
    ];
    io:println(j); // @output [{"name":"James","age":100,"children":["Jack","Jane"],"married":true}]
    if j is J[] {
        io:println("array"); // @output array
        J j0 = j[0];
        if j0 is map<J> {
            io:println("map"); // @output map
            io:println(j0["age"]); // @output 100
        }
    }
}
