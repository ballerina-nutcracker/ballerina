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

type Person record {|
    @constraint:Int {minValue: {value: 18, message: "age must be an adult."}}
    int age;
    @constraint:Float {maxFractionDigits: 2}
    float height;
    @constraint:String {minLength: 3, maxLength: {value: 8, message: "name is too long"}}
    string name;
    @constraint:String {minLength: 2}
    string nickname;
    @constraint:Array {maxLength: 2}
    string[] tags?;
|};

type Date record {|
    int year;
    int month;
    int day;
|};

type Event record {|
    string title;
    @constraint:Date {option: constraint:FUTURE}
    Date begins;
|};

@constraint:Date
type Holiday record {|
    string name;
    int year;
    int month;
    int day;
|};

public function main() {
    show(constraint:validate({age: 30, height: 1.75, name: "alice", nickname: "nn"}, Person)); //@output ok: {"age":30,"height":1.75,"name":"alice","nickname":"nn"}
    show(constraint:validate({age: 30, height: 1.75, name: "alice", nickname: "al", tags: ["a", "b"]}, Person)); //@output ok: {"age":30,"height":1.75,"name":"alice","nickname":"al","tags":["a","b"]}
    show(constraint:validate({age: 17, height: 1.755, name: "al", nickname: "nn"}, Person)); //@output ValidationError: age must be an adult and Validation failed for '$.height:maxFractionDigits','$.name:minLength' constraint(s).
    show(constraint:validate({age: 30, height: 1.75, name: "alicealice", nickname: "nn"}, Person)); //@output ValidationError: name is too long.
    show(constraint:validate({age: 10, height: 1.75, name: "alicealice", nickname: "nn"}, Person)); //@output ValidationError: age must be an adult and name is too long.
    show(constraint:validate({age: 10, height: 1.755, name: "alicealice", nickname: "a", tags: ["a", "b", "c"]}, Person)); //@output ValidationError: age must be an adult, name is too long and Validation failed for '$.height:maxFractionDigits','$.nickname:minLength','$.tags:maxLength' constraint(s).
    show(constraint:validate({title: "launch", begins: {year: 2999, month: 1, day: 1}}, Event)); //@output ok: {"title":"launch","begins":{"year":2999,"month":1,"day":1}}
    show(constraint:validate({title: "launch", begins: {year: 1999, month: 1, day: 1}}, Event)); //@output ValidationError: Validation failed for '$.begins:futureDate' constraint(s).
    show(constraint:validate({name: "x", year: 2023, month: 2, day: 30}, Holiday)); //@output ValidationError: Validation failed for '$.day:validDate' constraint(s).
    show(constraint:validate({name: "x", year: 2023, month: 2, day: 28}, Holiday)); //@output ok: {"name":"x","year":2023,"month":2,"day":28}
}
