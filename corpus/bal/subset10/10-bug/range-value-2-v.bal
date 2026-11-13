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

final object:Iterable R = 1 ... 2;

isolated function upTo(int n) returns object:Iterable {
    return 0 ..< n;
}

// A range value can be stored, typed as object:Iterable and iterated by
// foreach more than once; each iteration covers the whole sequence.
public function main() {
    var r = 7 ... 8;
    foreach int i in r {
        io:println(i); // @output 7
                       // @output 8
    }
    foreach int i in r {
        io:println(i); // @output 7
                       // @output 8
    }
    foreach int i in (5 ..< 6) {
        io:println(i); // @output 5
    }
    object:Iterable it = 1 ..< 3;
    io:println(it is object:Iterable); // @output true
    io:println(R is object:Iterable); // @output true
    io:println(upTo(2) is object:Iterable); // @output true
}
