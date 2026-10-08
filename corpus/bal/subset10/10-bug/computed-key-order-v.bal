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

function key(string k) returns string {
    io:println("key ", k);
    return k;
}

function val(int v) returns int {
    io:println("val ", v);
    return v;
}

function keyOrError(boolean bad) returns string|error {
    if bad {
        return error("bad key");
    }
    return "z";
}

function build(boolean bad) returns map<int>|error {
    return {[check keyOrError(bad)]: 1};
}

public function main() {
    map<int> m = {[key("k1")]: val(1), a: val(2), [key("k2")]: val(3)};
    // @output key k1
    // @output val 1
    // @output val 2
    // @output key k2
    // @output val 3
    io:println(m); // @output {"a":2,"k1":1,"k2":3}
    io:println(build(false)); // @output {"z":1}
    io:println(build(true)); // @output error("bad key")
}
