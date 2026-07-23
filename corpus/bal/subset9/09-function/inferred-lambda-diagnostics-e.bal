// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

type UnaryInt function (int) returns int;
type BinaryInt function (int, int) returns int;
type StringResult function (int) returns string;
type StringIdentity function (string) returns string;
type IntOrStringIdentity UnaryInt|StringIdentity;
type CrossParamsOne function (int, string) returns int;
type CrossParamsTwo function (string, int) returns int;
type RestInt function (int...) returns int;

function acceptFunction(function fn) {
    _ = fn;
}

function consume(any first, any second) returns int {
    _ = first;
    _ = second;
    return 1;
}

public function main() {
    var noExpectedType = value => value; // @error
    acceptFunction(value => value); // @error

    BinaryInt wrongArity = value => value; // @error
    IntOrStringIdentity incompatibleArityTypes = value => value; // @error
    CrossParamsOne & CrossParamsTwo incompatibleParameters = (first, second) => consume(first, second); // @error

    StringResult incompatibleReturn = value => value; // @error
    RestInt incompatibleFunctionType = (first, second) => first + second; // @error

    _ = noExpectedType;
    _ = wrongArity;
    _ = incompatibleArityTypes;
    _ = incompatibleParameters;
    _ = incompatibleReturn;
    _ = incompatibleFunctionType;
}
