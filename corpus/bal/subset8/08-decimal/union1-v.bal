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
    decimal|int d1 = 1;
    io:println(d1); // @output 1
    io:println(d1 is int); // @output true

    d1 = 1.2;
    io:println(d1); // @output 1.2
    io:println(d1 is decimal); // @output true

    decimal|float d3 = 1.22d;
    io:println(d3); // @output 1.22
    io:println(d3 is decimal); // @output true

    d3 = 1.2;
    io:println(d3 is float); // @output true
}
