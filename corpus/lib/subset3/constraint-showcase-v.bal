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

@constraint:Int {minValue: 18, maxValue: 100}
type Age int;

@constraint:Array {minLength: 1, maxLength: 3}
type Names string[];

type User record {|
    @constraint:Int {minValue: 18}
    int age;
    @constraint:String {minLength: 3, maxLength: {value: 8, message: "name too long"}}
    string name;
    @constraint:Float {maxFractionDigits: 2}
    float score;
|};

type Date record {|
    int year;
    int month;
    int day;
|};

@constraint:Date {option: constraint:PAST}
type Birthday Date;

public function main() {
    User|error ok = constraint:validate({age: 30, name: "alice", score: 1.25}, User);
    io:println(ok is User); //@output true
    User|error bad = constraint:validate({age: 10, name: "al", score: 1.234}, User);
    io:println(bad is error ? bad.message() : "ok"); //@output Validation failed for '$.age:minValue','$.name:minLength','$.score:maxFractionDigits' constraint(s).
    io:println(constraint:validate(5, Age) is constraint:ValidationError); //@output true
    io:println(constraint:validate(["a", "b", "c", "d"], Names) is constraint:ValidationError); //@output true
    io:println(constraint:validate({year: 2999, month: 1, day: 1}, Birthday) is constraint:ValidationError); //@output true
    io:println(constraint:validate("text", Age) is constraint:TypeConversionError); //@output true
}
