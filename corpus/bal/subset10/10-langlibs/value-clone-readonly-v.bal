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

type Config readonly & record {|
    string host;
    int port;
|};

function add(int a, int b) returns int => a + b;

public function main() {
    Config cfg = {host: "localhost", port: 8080};
    io:println(cfg.clone() === cfg); // @output true

    readonly & int[] ids = [1, 2, 3];
    io:println(ids.clone() === ids); // @output true

    // Readonly subtrees of a mutable value are shared with the clone.
    map<anydata> holder = {"cfg": cfg, "ids": ids, "xs": [4, 5]};
    map<anydata> holderClone = holder.clone();
    io:println(holderClone === holder); // @output false
    io:println(holderClone["cfg"] === cfg); // @output true
    io:println(holderClone["ids"] === ids); // @output true
    io:println(holderClone["xs"] === holder["xs"]); // @output false

    // Readonly at runtime is enough; the static type need not say so.
    anydata widened = ids;
    io:println(widened.clone() === ids); // @output true

    error e = error("oops");
    io:println(e.clone() === e); // @output true

    // Function values are inherently immutable, so they are Cloneable and
    // their clone is the same value.
    function (int, int) returns int f = add;
    io:println(f.clone() === f); // @output true
    var lambda = function(int x) returns int => x * 2;
    io:println(lambda.clone() === lambda); // @output true
}
