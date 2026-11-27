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

// The static type of a configurable variable is its declared type intersected
// with readonly, so isolated functions can read it and it can be assigned to a
// readonly-typed variable. The list constructor gets readonly as its expected type.

import ballerina/io;

configurable int[] ports = [80, 443]; // @panic

isolated function count() returns int {
    return ports.length();
}

public function main() {
    int[] & readonly p = ports;
    io:println(count(), p);
}
