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

type Address record {|
    string city;
    int zip;
|};

type Customer record {|
    int id;
    string name;
    Address address;
    string[] tags;
|};

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

public function main() returns error? {
    int n = 60000 * scale();
    int total = 0;
    foreach int i in 0 ..< n {
        map<json> raw = {id: i, name: "c", address: {city: "x", zip: i % 1000}, tags: ["t1", "t2"]};
        Customer c = check raw.cloneWithType();
        total += c.address.zip + c.tags.length();
    }
    io:println(total); // @output 30090000
}
