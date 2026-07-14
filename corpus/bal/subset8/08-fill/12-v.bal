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

type T map<int>;

public function main() {
    T[] a = [];
    a[1] = {"a": 1};
    io:println(a); // @output [{},{"a":1}]

    T[3] b = [{"b": 0}];
    io:println(b); // @output [{"b":0},{},{}]

    T[][] c = [];
    c[1][1] = {"c": 10};
    io:println(c); // @output [[],[{},{"c":10}]]

    c[1][0]["tmp"] = 1;
    io:println(c); // @output [[],[{"tmp":1},{"c":10}]]
}

