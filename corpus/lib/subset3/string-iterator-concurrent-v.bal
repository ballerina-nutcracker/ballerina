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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/io;

public function main() {
    var iterator = "A".iterator();
    future<record {| string value; |}?> first = start iterator.next();
    future<record {| string value; |}?> second = start iterator.next();
    record {| string value; |}?|error firstResult = wait first;
    record {| string value; |}?|error secondResult = wait second;

    int resultCount = 0;
    string value = "";
    if firstResult is record {| string value; |} {
        resultCount += 1;
        value = firstResult.value;
    }
    if secondResult is record {| string value; |} {
        resultCount += 1;
        value = secondResult.value;
    }
    io:println(resultCount); // @output 1
    io:println(value); // @output A
}
