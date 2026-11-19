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

type F1 function () returns int?;
type F2 isolated function () returns int;

type O1 object {
    function next() returns int?;
};

type O2 object {
    isolated function next() returns int;
};

class C {
    isolated function next() returns int => 2;
}

isolated function one() returns int => 1;

isolated function callFn(F1 & F2 f) returns int? {
    return f();
}

isolated function callNext(O1 & O2 o) returns int? {
    return o.next();
}

// A function belonging to an intersection of function types has the qualifiers
// of every member type, so intersecting with an isolated function type gives
// an isolated function type.
public function main() {
    F1 & F2 f = one;
    io:println(callFn(f)); // @output 1
    io:println(f is isolated function); // @output true
    O1 & O2 o = new C();
    io:println(callNext(o)); // @output 2
}
