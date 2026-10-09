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

function flag(int i) returns boolean {
    return i % 2 == 0;
}

// A closure created under a condition or a match clause still adds its
// captures to the loop's group, so those variables are never narrowed anywhere
// in the loop.
public function main() {
    int|string x = 1;
    int|string y = 2;
    int i = 0;
    while i < 3 {
        if x is int {
            int v = x; // @error x is captured under the if below
            io:println(v);
        }
        if y is int {
            int v = y; // @error y is captured under the match below
            io:println(v);
        }
        if flag(i) {
            function () setX = function () {
                x = "x";
            };
            setX();
        }
        match i {
            1 => {
                function () setY = function () {
                    y = "y";
                };
                setY();
            }
        }
        i += 1;
    }
}
