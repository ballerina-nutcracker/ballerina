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
    float f = 99999999999999999999;
    io:println(f); // @output 1e20
    float g = 0x10000000000000000;
    io:println(g); // @output 1.8446744073709552e19
    decimal d = 99999999999999999999;
    io:println(d); // @output 99999999999999999999
    decimal dh = 0xFFFFFFFFFFFFFFFFFF;
    io:println(dh); // @output 4722366482869645213695
    int m = -9223372036854775808;
    io:println(m); // @output -9223372036854775808
    int n = 9223372036854775807;
    io:println(n); // @output 9223372036854775807
}
