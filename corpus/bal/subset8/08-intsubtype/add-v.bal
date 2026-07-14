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
    int:Unsigned32 a = 4294967295;
    io:println(a + a); //@output 8589934590
    int:Signed32 b = -2147483648;
    io:println(b + b); //@output -4294967296
    io:println(a + b); //@output 2147483647
    io:println(0 - a - a); //@output -8589934590
    io:println(-a - a); //@output -8589934590
}
