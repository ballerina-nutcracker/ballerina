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

import testorg/cross_compilation_unit_distinct_alias_v.basea;

type ChildB distinct ParentB;
type ParentA basea:BaseError;

public function main() {
    ChildA childA = error ChildA("child A");
    io:println(childA is ParentA); // @output true
    io:println(childA is ChildB); // @output false

    ChildB childB = error ChildB("child B");
    io:println(childB is ParentB); // @output true
    io:println(childB is ChildA); // @output false
}
