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

// A range expression that is not the collection of a foreach still evaluates
// to an object belonging to Iterable<int,()> (spec 6.26).
public function main() {
    var r = 1 ..< 3;
    io:println(r is object:Iterable); // @output true
    io:println((1 ... 3) is object:Iterable); // @output true
    any a = 1 ..< 2;
    io:println(a is object:Iterable); // @output true
}
