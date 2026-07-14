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
    float[][] vv = [
        [1, 2],
        [3, 4, 5, 0],
        [0.5],
        []
    ];
    float total = 0;
    foreach int i in 0 ..< vv.length() {
        float[] v = vv[i];
        foreach int j in 0 ..< v.length() {
            total += v[j];
        }
    }
    io:println(total); // @output 15.5
}

