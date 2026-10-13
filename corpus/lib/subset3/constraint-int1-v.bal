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

@constraint:Int {minValue: 18, maxValue: 100}
type Age int;

@constraint:Int {minValueExclusive: 0, maxValueExclusive: 10}
type Digit int;

@constraint:Int {maxDigits: 3}
type Short int;

@constraint:Int {minValue: {value: 1, message: "must be positive."}, maxValue: 5}
type Custom int;

public function main() {
    show(constraint:validate(30, Age)); //@output ok: 30
    show(constraint:validate(17, Age)); //@output ValidationError: Validation failed for '$:minValue' constraint(s).
    show(constraint:validate(101, Age)); //@output ValidationError: Validation failed for '$:maxValue' constraint(s).
    show(constraint:validate(18, Age)); //@output ok: 18
    show(constraint:validate(100, Age)); //@output ok: 100
    show(constraint:validate(0, Digit)); //@output ValidationError: Validation failed for '$:minValueExclusive' constraint(s).
    show(constraint:validate(10, Digit)); //@output ValidationError: Validation failed for '$:maxValueExclusive' constraint(s).
    show(constraint:validate(5, Digit)); //@output ok: 5
    show(constraint:validate(999, Short)); //@output ok: 999
    show(constraint:validate(1000, Short)); //@output ValidationError: Validation failed for '$:maxDigits' constraint(s).
    show(constraint:validate(-999, Short)); //@output ok: -999
    show(constraint:validate(-1000, Short)); //@output ValidationError: Validation failed for '$:maxDigits' constraint(s).
    show(constraint:validate(0, Custom)); //@output ValidationError: must be positive.
    show(constraint:validate(6, Custom)); //@output ValidationError: Validation failed for '$:maxValue' constraint(s).
    show(constraint:validate(3, Custom)); //@output ok: 3
}
