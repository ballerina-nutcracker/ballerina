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

// The iterator of a range value yields each integer in increasing order and
// then nil; empty ranges yield nil immediately, and ranges at the ends of the
// int range stop without overflowing.
public function main() {
    var r = 1 ... 2;
    var it = r.iterator();
    io:println(it.next()); // @output {"value":1}
    io:println(it.next()); // @output {"value":2}
    io:println(it.next() is ()); // @output true
    io:println(it.next() is ()); // @output true

    var empty = 3 ..< 3;
    io:println(empty.iterator().next() is ()); // @output true
    var reversed = 3 ... 2;
    io:println(reversed.iterator().next() is ()); // @output true

    var bottom = -9223372036854775807 - 1 ..< -9223372036854775807;
    foreach int i in bottom {
        io:println(i); // @output -9223372036854775808
    }
    var bottomEmpty = -9223372036854775807 - 1 ..< -9223372036854775807 - 1;
    io:println(bottomEmpty.iterator().next() is ()); // @output true

    var top = 9223372036854775806 ... 9223372036854775807;
    foreach int i in top {
        io:println(i); // @output 9223372036854775806
                       // @output 9223372036854775807
    }
}
