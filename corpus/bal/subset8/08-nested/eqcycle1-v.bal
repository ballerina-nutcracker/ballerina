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
    map<J> j1 = {loop: ()};
    map<J> j2 = {loop: ()};
    io:println(j1 == j2); // @output true
    j1["loop"] = j1;
    io:println(j1 == j2); // @output false
    j2["loop"] = j2;
    io:println(j1 == j2); // @output true
    j2["loop"] = j1;
    map<J> j3 = {loop: ()};
    j3["loop"] = {loop: {loop: {loop: j3}}};
    io:println(j1 == j3); // @output true
}
