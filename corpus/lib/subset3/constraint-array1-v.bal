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

@constraint:Array {minLength: 1, maxLength: 3}
type Names string[];

@constraint:Array {length: 2}
type Pair int[];

@constraint:Array {minLength: {value: 2, message: "need at least two."}}
type Custom int[];

public function main() {
    show(constraint:validate(["a"], Names)); //@output ok: ["a"]
    show(constraint:validate([], Names)); //@output ValidationError: Validation failed for '$:minLength' constraint(s).
    show(constraint:validate(["a", "b", "c", "d"], Names)); //@output ValidationError: Validation failed for '$:maxLength' constraint(s).
    show(constraint:validate([1, 2], Pair)); //@output ok: [1,2]
    show(constraint:validate([1, 2, 3], Pair)); //@output ValidationError: Validation failed for '$:length' constraint(s).
    show(constraint:validate([1], Custom)); //@output ValidationError: need at least two.
    show(constraint:validate([1, 2], Custom)); //@output ok: [1,2]
}
