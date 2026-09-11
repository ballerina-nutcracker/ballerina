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
import testorg/crossmoduleopaquedefaults.helper;
import testorg/crossmoduleopaquedefaults.types;

public function main() {
    types:Positions p = {};
    io:println(p.first); // @output 1
    io:println(p.fromThird); // @output 3
    io:println(p.named); // @output 2

    types:Positions q = {first: 7, fromThird: 8, named: 9};
    io:println(q.first); // @output 7
    io:println(q.fromThird); // @output 8
    io:println(q.named); // @output 9

    int[] values = [10, 20, 30, 20];
    io:println(helper:firstOf(values, 20)); // @output 1
    io:println(helper:firstOf(values, 30)); // @output 2
    io:println(helper:firstOf(values, 20, 99)); // @output 99
    io:println(helper:firstNamed(values)); // @output 1
    io:println(helper:firstQualified(values)); // @output 2
    io:println(helper:firstOf(arr = values, val = 20)); // @output 1
}
