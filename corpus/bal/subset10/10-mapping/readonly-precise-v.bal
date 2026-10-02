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

public function main() {
    var x = {readonly a: [1, 2], readonly inner: {v: 1}};
    int[] y = x.a;
    io:println(y); // @output [1,2]
    io:println(isReadonly(y)); // @output true
    io:println(x.inner.v); // @output 1
    io:println(isReadonly(x.inner)); // @output true
}

function isReadonly(any v) returns boolean {
    return v is readonly;
}
