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
    int[] arr = [10, 20, 30, 40, 30];

    int? first = arr.indexOf(30, 0);
    if first is int {
        io:println(first); // @output 2
    }

    int? firstDefault = arr.indexOf(30);
    if firstDefault is int {
        io:println(firstDefault); // @output 2
    }

    int? afterFirst = arr.indexOf(30, 3);
    if afterFirst is int {
        io:println(afterFirst); // @output 4
    }

    int? missing = arr.indexOf(99, 0);
    io:println(missing is ()); // @output true

    int? atLength = arr.indexOf(30, arr.length());
    io:println(atLength is ()); // @output true

    int? pastLength = arr.indexOf(30, arr.length() + 1000);
    io:println(pastLength is ()); // @output true

    int removed = arr.remove(1);
    io:println(removed); // @output 20
    io:println(arr.length()); // @output 4

    arr.removeAll();
    io:println(arr.length()); // @output 0
}
