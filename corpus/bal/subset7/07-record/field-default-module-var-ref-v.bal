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

// Skipped: https://github.com/ballerina-nutcracker/ballerina/issues/915
//
// This one runs and prints the right answer, but the reference to `base`
// never gets a determined type, so TestTypeResolver and TestSemanticAnalysis
// report "does not have determined type set" for it.
final int base = 5;

isolated function twice(int x) returns int => x * 2;

type Doubled record {|
    int n = twice(base);
|};

public function main() {
    Doubled d = {};
    io:println(d.n); // @output 10

    Doubled e = {n: 3};
    io:println(e.n); // @output 3
}
