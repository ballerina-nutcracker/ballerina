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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/io;

function number() returns int|error { return 1; }
function boolValue() returns boolean|error { return true; }
function failure() returns int|error { return error("failure"); }

function checkedOperands() returns int|error {
    int a = check boolValue() ? 1 : 2;
    int b = true ? check number() : 2;
    int c = false ? check failure() : check number();
    return a + b + c;
}

function selectedFailure() returns int|error {
    return true ? check failure() : 2;
}

public function main() returns error? {
    function() returns int|error fn = true ? function() returns int|error {
        return check number();
    } : function() returns int|error { return 0; };
    io:println(check checkedOperands()); // @output 3
    io:println(selectedFailure() is error); // @output true
    io:println(check fn()); // @output 1
}
