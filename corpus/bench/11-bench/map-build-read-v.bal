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
import ballerina/os;

function scale() returns int {
    string raw = os:getEnv("BAL_SCALE");
    if raw == "" {
        return 1;
    }
    int|error n = int:fromString(raw);
    if n is int && n > 0 {
        return n;
    }
    panic error("BAL_SCALE must be a positive integer: " + raw);
}

public function main() {
    int n = 200000 * scale();
    map<int> m = {};
    foreach int i in 0 ..< n {
        m["key-" + i.toString()] = i;
    }
    int total = 0;
    foreach int i in 0 ..< n {
        total += m.get("key-" + i.toString()) % 10;
    }
    foreach string k in m.keys() {
        if k.substring(k.length() - 1) == "7" {
            total += 1;
        }
    }
    io:println(m.length(), " ", total); // @output 200000 920000
}
