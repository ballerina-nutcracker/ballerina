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

// Removing members backed by a tuple's rest type stays legal; only the
// mandatory leading members are protected.
public function main() {
    [int, string, int...] t = [1, "a", 2, 3];
    io:println(t.remove(2)); // @output 2
    io:println(t); // @output [1,"a",3]

    [int, int...] u = [1, 2];
    io:println(u.remove(1)); // @output 2
    io:println(u); // @output [1]

    int[] open = [10, 20, 30];
    io:println(open.remove(1)); // @output 20
    open.removeAll();
    io:println(open.length()); // @output 0
}
