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

type Integer 1d|2d|3d;

public function main() {
    any v = 1d;
    io:println(v is Integer); // @output true
    v = 2d;
    io:println(v is Integer); // @output true
    v = 2.000d;
    io:println(v is Integer); // @output true
    v = 2;
    io:println(v is Integer); // @output false
    v = 3d;
    io:println(v is Integer); // @output true
    v = 4d;
    io:println(v is Integer); // @output false
}
