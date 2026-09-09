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

import ballerina/lang.array;
import ballerina/io;

// Parameter names are part of the public interface, so jBallerina accepts all
// of these. `array:indexOf` declares `int startIndex = 0`; the default is
// supplied by a compiler-generated provider shared by every monomorphic form,
// so it can be omitted positionally or by naming only the earlier parameters.
public function main() {
    int[] a = [1, 2, 3, 2];
    io:println(a.indexOf(2)); // @output 1
    io:println(a.indexOf(2, 2)); // @output 3
    io:println(a.indexOf(val = 2)); // @output 1
    io:println(a.indexOf(2, startIndex = 2)); // @output 3
    io:println(a.indexOf(val = 2, startIndex = 2)); // @output 3

    io:println(array:indexOf(a, 2)); // @output 1
    io:println(array:indexOf(a, 2, 2)); // @output 3
    io:println(array:indexOf(a, val = 2)); // @output 1
    io:println(array:indexOf(a, 2, startIndex = 2)); // @output 3
    io:println(array:indexOf(arr = a, val = 2, startIndex = 2)); // @output 3

    int[] b = [10, 20, 30];
    io:println(b.remove(index = 0)); // @output 10
    io:println(b); // @output [20,30]

    map<int> m = {"a": 1, "b": 2};
    io:println(m.remove(k = "a")); // @output 1
}
