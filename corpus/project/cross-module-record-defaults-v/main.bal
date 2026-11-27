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

import crossmodulerecorddefaults.helper;

import ballerina/io;

const helper:Derived A = {runtime: 3};

public function main() {
    io:println(A); // @output {"runtime":3,"flag":true,"x":1,"n":null,"s":"b","list":[1,2],"mapping":{"k":3},"nested":{"depth":1}}
    io:println(A is helper:Derived); // @output true
    io:println(A is readonly); // @output true
    io:println(A.nested.depth); // @output 1
    io:println(A.mapping["k"]); // @output 3
    helper:Derived runtimeValue = {};
    io:println(runtimeValue.runtime); // @output 42
}
