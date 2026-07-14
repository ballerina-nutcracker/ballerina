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
    byte x = 5;
    byte y = 3;
    byte z = 0;
    int i = x * y;
    int j = x * z;
    io:println(i); //@output 15
    io:println(j); //@output 0
    int:Signed32 a = 2147483647;
    int:Signed32 b = 2147483647;
    int c = a * b;
    io:println(c); //@output 4611686014132420609
    int:Signed32 d = -2147483647;
    int e = d * a;
    io:println(e); //@output -4611686014132420609
}
