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

// @productions float multiplicative-expr floating-point-literal return-stmt unary-expr function-call-expr local-var-decl-stmt
import ballerina/io;

public function main() {
    float nan = 0.0 / 0.0;
    float anotherNan = -0.0 / 0.0;
    float pInf = 1.0 / 0.0;
    float nInf = -1.0 / 0.0;

    io:println(nan % nan); // @output NaN
    io:println(nan % pInf); // @output NaN
    io:println(nan % 7.0); // @output NaN
    io:println(nan % 5.0); // @output NaN
    io:println(nan % 0.7); // @output NaN
    io:println(nan % 0.5); // @output NaN
    io:println(nan % 0.0); // @output NaN
    io:println(nan % -7.0); // @output NaN
    io:println(nan % -5.0); // @output NaN
    io:println(nan % -0.7); // @output NaN
    io:println(nan % -0.5); // @output NaN
    io:println(nan % -0.0); // @output NaN
    io:println(nan % nInf); // @output NaN
    io:println(nan % anotherNan); // @output NaN

    io:println(pInf % nan); // @output NaN
    io:println(7.0 % nan); // @output NaN
    io:println(5.0 % nan); // @output NaN
    io:println(0.7 % nan); // @output NaN
    io:println(0.5 % nan); // @output NaN
    io:println(0.0 % nan); // @output NaN
    io:println(-7.0 % nan); // @output NaN
    io:println(-5.0 % nan); // @output NaN
    io:println(-0.7 % nan); // @output NaN
    io:println(-0.5 % nan); // @output NaN
    io:println(-0.0 % nan); // @output NaN
    io:println(nInf % nan); // @output NaN
    io:println(anotherNan % nan); // @output NaN

    io:println(pInf % nan); // @output NaN
    io:println(pInf % pInf); // @output NaN
    io:println(pInf % 7.0); // @output NaN
    io:println(pInf % 5.0); // @output NaN
    io:println(pInf % 0.7); // @output NaN
    io:println(pInf % 0.5); // @output NaN
    io:println(pInf % 0.0); // @output NaN
    io:println(pInf % -7.0); // @output NaN
    io:println(pInf % -5.0); // @output NaN
    io:println(pInf % -0.7); // @output NaN
    io:println(pInf % -0.5); // @output NaN
    io:println(pInf % -0.0); // @output NaN
    io:println(pInf % nInf); // @output NaN
    io:println(pInf % anotherNan); // @output NaN

    io:println(nInf % nan); // @output NaN
    io:println(nInf % pInf); // @output NaN
    io:println(nInf % 7.0); // @output NaN
    io:println(nInf % 5.0); // @output NaN
    io:println(nInf % 0.7); // @output NaN
    io:println(nInf % 0.5); // @output NaN
    io:println(nInf % 0.0); // @output NaN
    io:println(nInf % -7.0); // @output NaN
    io:println(nInf % -5.0); // @output NaN
    io:println(nInf % -0.7); // @output NaN
    io:println(nInf % -0.5); // @output NaN
    io:println(nInf % -0.0); // @output NaN
    io:println(nInf % nInf); // @output NaN
    io:println(nInf % anotherNan); // @output NaN

    io:println(nan % 0.0); // @output NaN
    io:println(pInf % 0.0); // @output NaN
    io:println(7.0 % 0.0); // @output NaN
    io:println(5.0 % 0.0); // @output NaN
    io:println(0.7 % 0.0); // @output NaN
    io:println(0.5 % 0.0); // @output NaN
    io:println(0.0 % 0.0); // @output NaN
    io:println(-7.0 % 0.0); // @output NaN
    io:println(-5.0 % 0.0); // @output NaN
    io:println(-0.7 % 0.0); // @output NaN
    io:println(-0.5 % 0.0); // @output NaN
    io:println(-0.0 % 0.0); // @output NaN
    io:println(nInf % 0.0); // @output NaN
    io:println(anotherNan % 0.0); // @output NaN

    io:println(7.0 % pInf); // @output 7.0
    io:println(5.0 % pInf); // @output 5.0
    io:println(0.7 % pInf); // @output 0.7
    io:println(0.5 % pInf); // @output 0.5
    io:println(0.0 % pInf); // @output 0.0
    io:println(-7.0 % pInf); // @output -7.0
    io:println(-5.0 % pInf); // @output -5.0
    io:println(-0.7 % pInf); // @output -0.7
    io:println(-0.5 % pInf); // @output -0.5
    io:println(-0.0 % pInf); // @output -0.0

    io:println(7.0 % nInf); // @output 7.0
    io:println(5.0 % nInf); // @output 5.0
    io:println(0.7 % nInf); // @output 0.7
    io:println(0.5 % nInf); // @output 0.5
    io:println(0.0 % nInf); // @output 0.0
    io:println(-7.0 % nInf); // @output -7.0
    io:println(-5.0 % nInf); // @output -5.0
    io:println(-0.7 % nInf); // @output -0.7
    io:println(-0.5 % nInf); // @output -0.5
    io:println(-0.0 % nInf); // @output -0.0

    io:println(7.0 % 7.0); // @output 0.0
    io:println(5.0 % 7.0); // @output 5.0
    io:println(0.7 % 7.0); // @output 0.7
    io:println(0.5 % 7.0); // @output 0.5
    io:println(0.0 % 7.0); // @output 0.0
    io:println(-7.0 % 7.0); // @output -0.0
    io:println(-5.0 % 7.0); // @output -5.0
    io:println(-0.7 % 7.0); // @output -0.7
    io:println(-0.5 % 7.0); // @output -0.5
    io:println(-0.0 % 7.0); // @output -0.0

    io:println(7.0 % 5.0); // @output 2.0
    io:println(-7.0 % 5.0); // @output -2.0

    io:println(7.0 % -7.0); // @output 0.0
    io:println(5.0 % -7.0); // @output 5.0
    io:println(0.7 % -7.0); // @output 0.7
    io:println(0.5 % -7.0); // @output 0.5
    io:println(0.0 % -7.0); // @output 0.0
    io:println(-7.0 % -7.0); // @output -0.0
    io:println(-5.0 % -7.0); // @output -5.0
    io:println(-0.7 % -7.0); // @output -0.7
    io:println(-0.5 % -7.0); // @output -0.5
    io:println(-0.0 % -7.0); // @output -0.0

    io:println(7.0 % -5.0); // @output 2.0
    io:println(-7.0 % -5.0); // @output -2.0

    io:println(floatRem(1.0, 0.1)); // @output 0.09999999999999995
    float f = 100f;
    io:println(f % 19.0); // @output 5.0
    io:println(1999.0 % f); // @output 99.0
}

function floatRem(float f1, float f2) returns float {
    return f1 % f2;
}
