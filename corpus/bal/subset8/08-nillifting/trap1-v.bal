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
    int? left = 1;
    int? right = 2;
    var sum = trap left + right;
    io:println(sum); // @output 3

    var negated = trap -left;
    io:println(negated); // @output -1

    int? nilValue = ();
    var nilSum = trap left + nilValue;
    io:println(nilSum is ()); // @output true

    int? zero = 0;
    var divided = trap left / zero;
    io:println(divided); // @output error("divide by zero")
}
