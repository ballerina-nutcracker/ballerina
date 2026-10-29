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
    (function (int) returns int)[]|[string, string] hs = [(p) => p * 2];
    if hs is (function (int) returns int)[] {
        var h = hs[0];
        io:println(h(3)); // @output 6
    }
    (function (int) returns int)[]|(function (string) returns string)[] fs = [(p) => p * 2];
    if fs is (function (int) returns int)[] {
        var h = fs[0];
        io:println(h(3)); // @output 6
    }
}
