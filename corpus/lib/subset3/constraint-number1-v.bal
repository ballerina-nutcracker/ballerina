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

import ballerina/constraint;
import ballerina/io;

function show(anydata|error r) {
    if r is constraint:ValidationError {
        io:println("ValidationError: ", r.message());
    } else if r is constraint:TypeConversionError {
        io:println("TypeConversionError: ", r.message());
    } else if r is constraint:Error {
        io:println("Error: ", r.message());
    } else if r is error {
        io:println("error: ", r.message());
    } else {
        io:println("ok: ", r.toString());
    }
}

@constraint:Number {minValue: 1.5, maxValue: 3.5, maxIntegerDigits: 1, maxFractionDigits: 1}
type Level decimal;

@constraint:Number {minValueExclusive: 0, maxValueExclusive: 10}
type Score int|float|decimal;

public function main() {
    show(constraint:validate(2.5d, Level)); //@output ok: 2.5
    show(constraint:validate(2.55d, Level)); //@output ValidationError: Validation failed for '$:maxFractionDigits' constraint(s).
    show(constraint:validate(12.55d, Level)); //@output ValidationError: Validation failed for '$:maxFractionDigits','$:maxIntegerDigits','$:maxValue' constraint(s).
    show(constraint:validate(0.5d, Level)); //@output ValidationError: Validation failed for '$:minValue' constraint(s).
    show(constraint:validate(3.5d, Level)); //@output ok: 3.5
    show(constraint:validate(0, Score)); //@output ValidationError: Validation failed for '$:minValueExclusive' constraint(s).
    show(constraint:validate(5, Score)); //@output ok: 5
    show(constraint:validate(10.0, Score)); //@output ValidationError: Validation failed for '$:maxValueExclusive' constraint(s).
    show(constraint:validate(9.99d, Score)); //@output ok: 9.99
    show(constraint:validate(-1.5, Score)); //@output ValidationError: Validation failed for '$:minValueExclusive' constraint(s).
}
