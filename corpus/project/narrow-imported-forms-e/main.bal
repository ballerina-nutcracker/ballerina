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
import testorg/narrow_imported_forms_e.data;

// Neither a final module variable nor a function of an imported module can be
// narrowed, whatever the form of the narrowing.
public function main() {
    if data:pinned is int {
    } else {
        string s = data:pinned; // @error an imported final module variable cannot be narrowed
        io:println(s);
    }
    if data:pinned == 2 {
        2 two = data:pinned; // @error an imported final module variable cannot be narrowed
        io:println(two);
    }
    match data:pinned {
        2 => {
            2 matched = data:pinned; // @error an imported final module variable cannot be narrowed
            io:println(matched);
        }
    }
    if data:convert is function (int) returns int {
        function (int) returns int f = data:convert; // @error an imported module function cannot be narrowed
        io:println(f(1));
    }
}
