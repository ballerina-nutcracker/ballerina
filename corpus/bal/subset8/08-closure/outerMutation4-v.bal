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

type F function (float) returns float;

function runner(F f, float val) returns float {
    return f(val);
}

public function main() {
    float v1 = 10.0;
    float v2 = 20.0;
    float res = v1 + runner(function(float a) returns float {
                v1 += a;
                return 5;
            }, 2) + v1 + v2 + runner(function(float b) returns float {
                float _ = b;
                v2 += v1;
                return 0;
            }, 2);
    io:println(res); // @output 47.0
    io:println(v1); // @output 12.0
    io:println(v2); // @output 32.0
}
