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
    int[4] i5 = [1, 2, 3, 4];
    io:println(i5); //@output [1,2,3,4]
    io:println(i5[1]); //@output 2
    i5[2] = 17;
    io:println(i5); //@output [1,2,17,4]
    i5 = [4, 5, 6, 7];
    io:println(i5); //@output [4,5,6,7]
    boolean[3] b3 = [true, false, true];
    io:println(b3); //@output [true,false,true]
    io:println(b3[1]); //@output false
    b3[1] = true;
    io:println(b3); //@output [true,true,true]
    int[0] i0 = [];
    io:println(i0); //@output []
    float[3] f3 = [1.5, 2.5, 3.5];
    f3[1] += 1.0;
    io:println(f3); //@output [1.5,3.5,3.5]
}
