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

// A default in the function type of a record field is constructed when the record
// type descriptor is evaluated. Unlike a record field default, it is not isolated.
// jBallerina accepts this narrowing: https://github.com/ballerina-platform/ballerina-lang/issues/44765
public function main() {
    int|string x = 1;
    record {| function (int a = x is int ? 1 : 2) returns int f; |} r = {f: id};
    function (int a) returns int f = r.f;
    if x is int {
        int y = x; // @error x is captured by the default, so it is no longer narrowed to int
        io:println(y + f(0));
    }
}

function id(int a) returns int {
    return a;
}
