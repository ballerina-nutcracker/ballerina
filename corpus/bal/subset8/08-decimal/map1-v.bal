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
    map<decimal> m1 = {"x1": 2, "x2": 2.3, "x3": 2.3e34d};
    io:println(m1); // @output {"x1":2,"x2":2.3,"x3":2.3E+34}

    map<decimal|int> m2 = {"x1": 1.2};
    io:println(m2["x1"] is decimal); // @output true

    m2["x2"] = 23e34d;
    io:println(m2); // @output {"x1":1.2,"x2":2.3E+35}
}
