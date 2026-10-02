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

@constraint:String {length: 3}
type Code string;

@constraint:String {minLength: 2, maxLength: {value: 4, message: "name is too long!"}}
type Name string;

public function main() {
    show(constraint:validate("abc", Code)); //@output ok: abc
    show(constraint:validate("abcd", Code)); //@output ValidationError: Validation failed for '$:length' constraint(s).
    show(constraint:validate("ab", Code)); //@output ValidationError: Validation failed for '$:length' constraint(s).
    show(constraint:validate("a😀", Code)); //@output ok: a😀
    show(constraint:validate("😀", Code)); //@output ValidationError: Validation failed for '$:length' constraint(s).
    show(constraint:validate("a", Name)); //@output ValidationError: Validation failed for '$:minLength' constraint(s).
    show(constraint:validate("ab", Name)); //@output ok: ab
    show(constraint:validate("abcde", Name)); //@output ValidationError: name is too long!.
    show(constraint:validate("é😀é", Name)); //@output ok: é😀é
}
