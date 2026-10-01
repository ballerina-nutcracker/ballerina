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

// A map expected type gives the multiple receive a map, as it would a mapping
// constructor, so the result takes keys beyond the received fields.
public function main() {
    worker a {
        1 -> function;
        3 -> function;
    }
    worker b {
        2 -> function;
    }
    map<int> m = <- {a, second: b};
    m["c"] = 3;
    io:println(m); // @output {"a":1,"second":2,"c":3}

    record {|int a; int...;|} r = <- {a};
    r["x"] = 4;
    io:println(r); // @output {"a":3,"x":4}
}
