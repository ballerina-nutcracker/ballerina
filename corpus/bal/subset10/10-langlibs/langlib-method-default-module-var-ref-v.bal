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

// The langlib method receiver is a module-level variable referenced from a
// parameter default, so the variable's type must be resolved before the
// method call in the default can be looked up.
function padded(string s = name.substring(0, 3)) returns string {
    return s;
}

function sized(int n = items.length()) returns int {
    return n;
}

// A preceding parameter used as a langlib method receiver in a default keeps
// working alongside the module-variable form.
function head(string text, int n = text.length()) returns string {
    return text.substring(0, n);
}

string name = "ballerina";
int[] items = [1, 2, 3, 4];

public function main() {
    io:println(padded()); // @output bal
    io:println(sized()); // @output 4
    io:println(head("abc")); // @output abc
    io:println(head("abcdef", 2)); // @output ab
}
