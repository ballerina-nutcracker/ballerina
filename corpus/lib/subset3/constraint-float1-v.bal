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

@constraint:Float {minValue: 0.5, maxValue: 9.5}
type Ratio float;

@constraint:Float {minValueExclusive: 0.0, maxValueExclusive: 1.0}
type Fraction float;

@constraint:Float {maxIntegerDigits: 2, maxFractionDigits: 2}
type Money float;

public function main() {
    show(constraint:validate(1.0, Ratio)); //@output ok: 1.0
    show(constraint:validate(0.1, Ratio)); //@output ValidationError: Validation failed for '$:minValue' constraint(s).
    show(constraint:validate(9.6, Ratio)); //@output ValidationError: Validation failed for '$:maxValue' constraint(s).
    show(constraint:validate(0.0, Fraction)); //@output ValidationError: Validation failed for '$:minValueExclusive' constraint(s).
    show(constraint:validate(1.0, Fraction)); //@output ValidationError: Validation failed for '$:maxValueExclusive' constraint(s).
    show(constraint:validate(0.25, Fraction)); //@output ok: 0.25
    show(constraint:validate(12.5, Money)); //@output ok: 12.5
    show(constraint:validate(123.456, Money)); //@output ValidationError: Validation failed for '$:maxFractionDigits','$:maxIntegerDigits' constraint(s).
    show(constraint:validate(100.0, Money)); //@output ValidationError: Validation failed for '$:maxIntegerDigits' constraint(s).
    show(constraint:validate(1.005, Money)); //@output ValidationError: Validation failed for '$:maxFractionDigits' constraint(s).
    show(constraint:validate(-12.34, Money)); //@output ok: -12.34
    show(constraint:validate(-0.001, Money)); //@output ValidationError: Validation failed for '$:maxFractionDigits' constraint(s).
    show(constraint:validate(1.5E-5, Money)); //@output ValidationError: Validation failed for '$:maxFractionDigits' constraint(s).
    show(constraint:validate(1234567.125, Money)); //@output ValidationError: Validation failed for '$:maxFractionDigits','$:maxIntegerDigits' constraint(s).
    show(constraint:validate(0.0, Money)); //@output ok: 0.0
}
