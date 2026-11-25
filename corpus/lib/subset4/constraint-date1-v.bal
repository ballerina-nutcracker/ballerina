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

type Date record {
    int year;
    int month;
    int day;
};

@constraint:Date {option: constraint:PAST}
type Past Date;

@constraint:Date {option: constraint:PAST_OR_PRESENT}
type PastOrPresent Date;

@constraint:Date {option: constraint:FUTURE, message: "invalid date"}
type Future Date;

@constraint:Date {option: constraint:FUTURE_OR_PRESENT}
type FutureOrPresent Date;

@constraint:Date
type Valid Date;

@constraint:Date {option: {value: constraint:PAST, message: "birthday must be in the past."}, message: "birthday is not a valid date"}
type Birthday Date;

public function main() {
    show(constraint:validate({year: 1970, month: 1, day: 1}, Past)); //@output ok: {"year":1970,"month":1,"day":1}
    show(constraint:validate({year: 2999, month: 1, day: 1}, Past)); //@output ValidationError: Validation failed for '$:pastDate' constraint(s).
    show(constraint:validate({year: 1970, month: 1, day: 1}, PastOrPresent)); //@output ok: {"year":1970,"month":1,"day":1}
    show(constraint:validate({year: 2999, month: 12, day: 31}, PastOrPresent)); //@output ValidationError: Validation failed for '$:pastOrPresentDate' constraint(s).
    show(constraint:validate({year: 2999, month: 1, day: 1}, Future)); //@output ok: {"year":2999,"month":1,"day":1}
    show(constraint:validate({year: 1970, month: 1, day: 1}, Future)); //@output ValidationError: Validation failed for '$:futureDate' constraint(s).
    show(constraint:validate({year: 1970, month: 2, day: 30}, Future)); //@output ValidationError: invalid date.
    show(constraint:validate({year: 2999, month: 1, day: 1}, FutureOrPresent)); //@output ok: {"year":2999,"month":1,"day":1}
    show(constraint:validate({year: 1970, month: 1, day: 1}, FutureOrPresent)); //@output ValidationError: Validation failed for '$:futureOrPresentDate' constraint(s).
    show(constraint:validate({year: 2024, month: 2, day: 29}, Valid)); //@output ok: {"year":2024,"month":2,"day":29}
    show(constraint:validate({year: 2023, month: 2, day: 29}, Valid)); //@output ValidationError: Validation failed for '$.day:validDate' constraint(s).
    show(constraint:validate({year: 1900, month: 2, day: 29}, Valid)); //@output ValidationError: Validation failed for '$.day:validDate' constraint(s).
    show(constraint:validate({year: 2000, month: 2, day: 29}, Valid)); //@output ok: {"year":2000,"month":2,"day":29}
    show(constraint:validate({year: 2023, month: 4, day: 31}, Valid)); //@output ValidationError: Validation failed for '$.day:validDate' constraint(s).
    show(constraint:validate({year: 2023, month: 13, day: 1}, Valid)); //@output ValidationError: Validation failed for '$.month:validDate' constraint(s).
    show(constraint:validate({year: 2023, month: 0, day: 32}, Valid)); //@output ValidationError: Validation failed for '$.day:validDate','$.month:validDate' constraint(s).
    show(constraint:validate({year: 1000000000, month: 1, day: 1}, Valid)); //@output ValidationError: Validation failed for '$.year:validDate' constraint(s).
    show(constraint:validate({year: -999999999, month: 1, day: 1}, Valid)); //@output ok: {"year":-999999999,"month":1,"day":1}
    show(constraint:validate({year: 0, month: 1, day: 1}, Valid)); //@output ok: {"year":0,"month":1,"day":1}
    show(constraint:validate({year: 2999, month: 1, day: 1}, Birthday)); //@output ValidationError: birthday must be in the past.
    show(constraint:validate({year: 2023, month: 2, day: 30}, Birthday)); //@output ValidationError: birthday is not a valid date.
    show(constraint:validate({year: 1990, month: 5, day: 17}, Birthday)); //@output ok: {"year":1990,"month":5,"day":17}
}
