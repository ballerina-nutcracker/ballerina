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

type Numbers [int, float, decimal, int...];

public function main() {
    Numbers nums = [1, 2, 3, 4, 5];
    int n = nums[0] + nums[3] + nums[4];
    io:println(n); // @output 10
    float f = nums[1] + 0.5;
    io:println(f); // @output 2.5
    decimal d = nums[2] + 0.5d;
    io:println(d); // @output 3.5
}
