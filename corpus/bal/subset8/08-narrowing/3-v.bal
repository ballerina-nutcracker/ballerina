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

// @productions float string string-literal exact-equality equality if-else-stmt equality-expr floating-point-literal return-stmt unary-expr function-call-expr
import ballerina/io;

public function main() {
    io:println(zero(0.0)); // @output +
    io:println(zero(-0.0)); // @output -
    io:println(zero(1.0)); // @output !
}

function zero(float v) returns string {
    if v == 0.0 {
        if v === -0.0 {
            return "-";
        }
        else if v === 0.0 {
            return "+";
        }
    }
    return "!";
}

