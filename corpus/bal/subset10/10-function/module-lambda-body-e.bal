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

// The body of a lambda in a module-level variable initializer is analyzed like
// the body of a lambda inside a function.
function () returns int moduleFn = function () returns int {
    int x = "s"; // @error a string is not an int
    return x;
};

isolated int counter = 0;

function () returns int readsCounter = function () returns int {
    return counter; // @error an isolated variable is read outside a lock
};

public function main() {
    io:println(moduleFn() + readsCounter());
}
