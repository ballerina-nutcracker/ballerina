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

// Merging the two branches recreates the entry that records the assignment, so
// the merge has to carry the positions from both branches for the loop
// diagnostic to report each assignment.
public function main() {
    any n = 1;
    if n is int {
        int i = 0;
        while i == 0 {
            i = 1;
            if i == 1 {
                n = 1; // @error cannot assign to a variable narrowed outside the enclosing loop
            } else {
                n = 2; // @error cannot assign to a variable narrowed outside the enclosing loop
            }
        }
        io:println(n);
    }
}
