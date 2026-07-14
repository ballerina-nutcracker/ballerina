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

// @productions float multiplicative-expr floating-point-literal return-stmt unary-expr function-call-expr assign-stmt local-var-decl-stmt

import ballerina/io;

public function main() {
    io:println(floatMul(3.0, 2.0)); // @output 6.0
    io:println(floatMul(1.0, 0.0)); // @output 0.0
    io:println(floatMul(0.0, -1.0)); // @output -0.0
    io:println(floatMul(0.0, -0.0)); // @output -0.0
    io:println(floatMul(0.0 / 0.0, 1.0)); // @output NaN
    io:println(floatMul(0.0 / 0.0, 0.0 / 0.0)); // @output NaN
    io:println(floatMul(1.0 / 0.0, 20f)); // @output Infinity
    io:println(floatMul(-1.0 / 0.0, 1.0 / 0.0)); // @output -Infinity
    io:println(floatMul(-1.0 / 0.0, 0f)); // @output NaN

    io:println(3.0 * 2.0); // @output 6.0
    io:println(1.0 * 0.0); // @output 0.0
    io:println(0.0 * -1.0); // @output -0.0
    io:println(0.0 * -0.0); // @output -0.0
    io:println(0.0 / 0.0 * 1.0); // @output NaN
    io:println(0.0 / 0.0 * 0.0 / 0.0); // @output NaN
    io:println(1.0 / 0.0 * 20f); // @output Infinity
    io:println(-1.0 / 0.0 * 1.0 / 0.0); // @output -Infinity
    io:println(-1.0 / 0.0 * 0f); // @output NaN

    float f1 = 2.0;
    io:println(21.0 * f1); // @output 42.0
    float f2 = 21.21;
    io:println(f1 * f2); // @output 42.42
    f2 = -1.0 / 0.0;
    io:println(f2 * 2f); // @output -Infinity
}

function floatMul(float f1, float f2) returns float {
    return f1 * f2;
}
