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

type R record {|
    readonly int a;
    int...;
|};

function isReadonly(any v) returns boolean {
    return v is readonly;
}

public function main() {
    string k = "a";
    readonly & map<int> m = {[k]: 1};
    io:println(m, isReadonly(m)); // @output {"a":1}true
    readonly & map<int[]> ra = {[k]: [1, 2]};
    io:println(isReadonly(ra[k])); // @output true
    readonly & R r = {a: 1, [k]: 2};
    io:println(r.a, isReadonly(r)); // @output 2true
}
