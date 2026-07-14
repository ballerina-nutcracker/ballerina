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
    int|boolean count = 3;
    while count is int {
        int i = count;
        io:println(i); // @output 3
        // @output 2
        // @output 1
        if count > 1 {
            count = count - 1;
        }
        else {
            count = false;
        }
    }
    if count is int {
        // Should be removed if #ballerina-spec/1019 is implemented.
        io:println("unreached");
    }
}
