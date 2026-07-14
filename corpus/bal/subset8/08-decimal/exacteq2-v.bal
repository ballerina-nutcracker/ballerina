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
    decimal d1 = 0d;
    decimal d2 = 0d;
    io:println(d1 === d2); // @output true
    d1 = -9.999999999999999999999999999999999E6144d;
    d2 = -9.999999999999999999999999999999999E6144d;
    io:println(d1 === d2); // @output true
    d1 = 0.000001d;
    d2 = 0.000001d;
    io:println(d1 === d2); // @output true
    d1 = 1.0d;
    d2 = 1.00d;
    io:println(d1 === d2); // @output false
    d1 = 0.00000100d;
    d2 = 0.000001d;
    io:println(d1 === d2); // @output false
    d1 = 1E-6142d;
    d2 = 10E-6143d;
    io:println(d1 === d2); // @output false
    d1 = 100E3d;
    d2 = 10E4d;
    io:println(d1 !== d2); // @output true
}
