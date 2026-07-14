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
    int? a = 4;
    int? b = 2;
    int? c = 1;

    int? v1 = a | b;
    io:println(v1); // @output 6
    io:println(a | b | c); // @output 7

    int? d = ();
    int? v2 = a | d;
    io:println(v2); // @output 
    io:println(a | b | c | d); // @output 

    int? v3 = c << b;
    io:println(v3); // @output 4
}
