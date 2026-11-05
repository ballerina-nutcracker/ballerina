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
    string key = "id";
    map<int> afterSpecific = {id: 1, [key]: 2};
    io:println(afterSpecific["id"]); // @output 2
    map<int> beforeSpecific = {[key]: 3, id: 1};
    io:println(beforeSpecific["id"]); // @output 3

    map<int> seed = {id: 1};
    map<int> afterSpread = {...seed, [key]: 4};
    io:println(afterSpread["id"]); // @output 4
    map<int> beforeSpread = {[key]: 5, ...seed};
    io:println(beforeSpread["id"]); // @output 5

    var inferred = {id: 1, [key]: 9};
    io:println(inferred.id); // @output 9

    map<int> repeated = {[key]: 6, [key]: 7, [key]: 8};
    io:println(repeated["id"]); // @output 8
}
