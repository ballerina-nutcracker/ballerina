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

type Holder record {|
    map<int> m?;
|};

public function main() returns error? {
    anydata v = {};
    map<map<int>> m = check v.cloneWithType();
    m["a"]["b"] = 1;
    io:println(m); // @output {"a":{"b":1}}

    Holder r = check v.cloneWithType();
    r.m["k"] = 1;
    io:println(r); // @output {"m":{"k":1}}

    anydata nested = {"a": {}};
    map<map<map<int>>> mm = check nested.cloneWithType();
    mm["a"]["b"]["c"] = 1;
    io:println(mm); // @output {"a":{"b":{"c":1}}}
}
