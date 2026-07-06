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

// @productions float function-call-expr method-call-expr local-var-decl-stmt lang-float-combined
import ballerina/io;

public function main() {
    // hypotenuse: sqrt(3^2 + 4^2) = 5
    float h = ((3.0).pow(2.0) + (4.0).pow(2.0)).sqrt();
    io:println(h);  // @output 5.0

    // ceiling and floor are duals: ceiling(-x) == -(floor(x))
    io:println((-2.3).ceiling() == -((2.3).floor()));  // @output true

    // abs of floor/ceiling
    io:println(((-3.7).floor()).abs());    // @output 4.0
    io:println(((-3.7).ceiling()).abs());  // @output 3.0

    // round is idempotent on integers
    float x = 5.0;
    io:println(x.round() == x);  // @output true
}
