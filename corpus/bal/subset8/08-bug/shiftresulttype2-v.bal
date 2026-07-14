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
    int:Unsigned8 a = 255;
    int:Unsigned8 b = 63;
    int:Unsigned8 c = 64;
    int:Unsigned8 d = 122;

    io:println(a << b); // @output -9223372036854775808

    io:println(a << c); // @output 255

    io:println(a << d); // @output -288230376151711744

    io:println(b << b); // @output -9223372036854775808

    io:println(b << c); // @output 63

    io:println(a << a); // @output -9223372036854775808
}
