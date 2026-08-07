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
    // map self-cycle: m["self"] = m
    map<anydata> m = {};
    anydata cyclic = m;
    m["self"] = cyclic;
    anydata|error r1 = trap cyclic.cloneWithType(anydata);
    io:println(r1 is error); // @output true

    // array self-cycle: arr[0] = arr
    anydata[] arr = [];
    anydata cyclicArr = arr;
    arr.push(cyclicArr);
    anydata|error r2 = trap cyclicArr.cloneWithType(anydata);
    io:println(r2 is error); // @output true

    // mutual cycle: m1.next = m2, m2.prev = m1
    map<anydata> m1 = {};
    map<anydata> m2 = {};
    anydata c1 = m1;
    anydata c2 = m2;
    m1["next"] = c2;
    m2["prev"] = c1;
    anydata|error r3 = trap c1.cloneWithType(anydata);
    io:println(r3 is error); // @output true

    // DAG (not a cycle): same value referenced from two branches — must succeed
    map<anydata> shared = {x: 1};
    anydata dag = {a: shared, b: shared};
    anydata|error r4 = trap dag.cloneWithType(anydata);
    io:println(r4 is error); // @output false
}
