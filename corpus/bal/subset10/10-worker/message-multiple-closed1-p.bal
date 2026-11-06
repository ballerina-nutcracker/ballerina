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

function addField(map<anydata> m) {
    m["c"] = 3; // @panic the record has no field c
}

// The multiple receive builds its record with the closed inherent type of the
// expected record, as a mapping constructor would, so it rejects another field.
public function main() {
    worker a {
        1 -> function;
    }
    worker b {
        2 -> function;
    }
    record {|int a; int b;|} r = <- {a, b};
    io:println(r); // @output {"a":1,"b":2}
    addField(r);
}
