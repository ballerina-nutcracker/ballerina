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

// @productions float string string-literal equality multiplicative-expr if-else-stmt equality-expr floating-point-literal relational-expr return-stmt unary-expr function-call-expr
import ballerina/io;

public function main() {
    io:println(floatCmp(1.0, 1.0)); // @output eq

    io:println(floatCmp(1.0, 2.0)); // @output lt
    io:println(floatCmp(2.0, 1.0)); // @output gt
    io:println(floatCmp(-1.0, 1.0)); // @output lt
    io:println(floatCmp(-0.5, -1.0)); // @output gt

    io:println(floatCmp(-0.5, 1.0 / 0.0)); // @output lt
    io:println(floatCmp(-0.5, -1.0 / 0.0)); // @output gt

    io:println(floatCmp(-0.0, 0.0)); // @output eq

    io:println(floatCmp(0.0, 0.0 / 0.0)); // @output one nan
    io:println(floatCmp(0.0 / 0.0, 0.0)); // @output one nan
    io:println(floatCmp(0.0 / 0.0, 0.0 / 0.0)); // @output both nan
}

function floatCmp(float f1, float f2) returns string {
    if f1 < f2 {
        if !(f1 > f2) {
            if f1 <= f1 {
                return "lt";
            }
            else {
                return "lt error 1";
            }
        }
        else {
            return "lt error 2";
        }
    }
    if f1 > f2 {
        if !(f1 < f2) {
            if f1 >= f1 {
                return "gt";
            }
            else {
                return "gt error 1";
            }
        }
        else {
            return "gt error 2";
        }
    }
    if f1 == f2 {
        if f1 == 0.0 / 0.0 {
            return "both nan";
        }

        if f1 <= f1 {
            if f1 >= f1 {
                return "eq";
            }
            else {
                return "eq error 1";
            }
        }
        else {
            return "eq error 2";
        }
    }
    return "one nan";
}
