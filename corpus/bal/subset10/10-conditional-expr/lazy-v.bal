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

int calls = 0;
function mark(boolean value) returns boolean {
    calls += 1;
    return value;
}

public function main() {
    boolean a = (true ? true : mark(false)) && mark(true);
    boolean b = (false ? true : false) && (true ? mark(true) : mark(false));
    boolean c = (true ? false : true) || (false ? mark(false) : mark(true));
    boolean d = true || (true ? mark(false) : mark(false));
    io:println(a); // @output true
    io:println(b); // @output false
    io:println(c); // @output true
    io:println(d); // @output true
    io:println(calls); // @output 2
}
