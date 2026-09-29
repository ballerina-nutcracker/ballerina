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
    int|string x = 1;
    if x is int {
        from int i in [1, 2] do {
            if i == 2 {
                x = "a";
            }
        };
        io:println(x is string); // @output true
        if x is int {
            io:println(x + 1);
        } else {
            io:println(x); // @output a
        }
    }

    int|string y = 5;
    if y is int {
        from int i in [1] do {
            _ = i;
        };
        io:println(y + 1); // @output 6
    }
}
