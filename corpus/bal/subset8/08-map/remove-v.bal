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
import ballerina/lang.map;

public function main() {
    map<int> scores = {alice: 10, bob: 20};
    int alice = map:remove(scores, "alice");
    io:println(alice); // @output 10
    io:println(scores.length()); // @output 1

    map<int> counts = {one: 1, two: 2};
    int two = map:remove(counts, "two");
    io:println(two); // @output 2
    io:println(counts.length()); // @output 1

    map<string> labels = {id: "A", name: "Ann"};
    string id = map:remove(labels, "id");
    io:println(id); // @output A
    io:println(labels.length()); // @output 1
}
