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

type C record {|
    float x;
    float y;
|};

const N = 10;

public function main() {
    C[] v = [];
    foreach int _ in 0 ..< N {
        v.push({x: 1, y: 2});
    }
    foreach int i in 0 ..< N {
        C c = v[i];
        c.x *= <float>i;
        c.y *= <float>i;
    }
    float total1 = 0;
    float total2 = 0;
    foreach int i in 0 ..< N {
        C c = v[i];
        total1 += c.x + c.y;
        total2 += 3.0 * <float>i;
    }
    io:println(total1 == total2); // @output true
}
