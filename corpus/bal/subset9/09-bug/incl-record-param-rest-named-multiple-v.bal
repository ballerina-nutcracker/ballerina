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

type StringConfig record {|
    string...;
|};

type IntConfig record {|
    never name?;
    never city?;
    int...;
|};

function foo(*StringConfig stringConfig, *IntConfig intConfig) {
    io:println(stringConfig["name"]); // @output Alice
    io:println(stringConfig["city"]); // @output New York
    io:println(intConfig["count"]); // @output 2
}

public function main() {
    foo(name = "Alice", city = "New York", count = 2);
}
