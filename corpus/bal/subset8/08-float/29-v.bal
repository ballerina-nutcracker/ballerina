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

// @productions float function-call-expr method-call-expr local-var-decl-stmt lang-float-round
import ballerina/io;

public function main() {
    io:println((2.5).round());    // @output 2.0
    io:println((3.5).round());    // @output 4.0
    io:println((-2.5).round());   // @output -2.0
    io:println((2.4).round());    // @output 2.0
    io:println((2.6).round());    // @output 3.0
    io:println((0.0).round());    // @output 0.0
    io:println((-0.0).round());   // @output -0.0
    float x = -0.5;
    float y = -0.4;
    io:println(x.round());       // @output -0.0
    io:println(y.round());       // @output -0.0
    io:println(1.0 / x.round()); // @output -Infinity
    io:println(1.0 / y.round()); // @output -Infinity
    io:println(x.round(0));      // @output -0.0
    io:println(y.round(0));      // @output -0.0
    io:println(1.0 / x.round(0)); // @output -Infinity
    io:println(1.0 / y.round(0)); // @output -Infinity
    io:println((0.5).round());   // @output 0.0
    io:println((0.4).round(0));  // @output 0.0
    io:println(1.0 / (0.5).round()); // @output Infinity
    io:println(1.0 / (0.4).round(0)); // @output Infinity
    io:println((1.0 / 0.0).round());   // @output Infinity
    io:println((-1.0 / 0.0).round());  // @output -Infinity
    io:println((0.0 / 0.0).round());   // @output NaN
}
