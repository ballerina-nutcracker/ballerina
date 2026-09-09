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



import ballerina/io;

const int AGE = 36;
const int[] NUMBERS = [1, 2];
const map<int> ANNOTATED = {readonly a: 1, b: 2};
const INFERRED = {readonly age: AGE, "name": 1};
const ALL_READONLY = {readonly a: 1};
const NESTED = {readonly inner: ALL_READONLY, other: 2};
const SHORTHAND = {readonly AGE};
const STRING_KEY = {readonly "age": AGE};
const CONST_REFS = {readonly numbers: NUMBERS, readonly values: ANNOTATED};

public function main() {
    io:println(ANNOTATED); // @output {"a":1,"b":2}
    io:println(INFERRED); // @output {"age":36,"name":1}
    io:println(ALL_READONLY); // @output {"a":1}
    io:println(NESTED); // @output {"inner":{"a":1},"other":2}
    io:println(SHORTHAND); // @output {"AGE":36}
    io:println(STRING_KEY); // @output {"age":36}
    io:println(CONST_REFS); // @output {"numbers":[1,2],"values":{"a":1,"b":2}}
}
