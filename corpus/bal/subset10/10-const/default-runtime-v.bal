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

isolated function trace(string name) returns int {
    io:println("default ", name);
    return 1;
}

type Traced record {|
    int a = trace("a");
    int b = trace("b");
|};

type Containers record {|
    map<int> m = {};
    int[] l = [];
    int x = 1;
|};

const Containers C = {};

public function main() {
    Traced first = {}; // @output default a
                       // @output default b
    io:println(first); // @output {"a":1,"b":1}
    Traced second = {a: 5}; // @output default b
    io:println(second); // @output {"a":5,"b":1}

    Containers one = {};
    Containers two = {};
    one.m["k"] = 1;
    one.l.push(9);
    io:println(one); // @output {"m":{"k":1},"l":[9],"x":1}
    io:println(two); // @output {"m":{},"l":[],"x":1}
    io:println(C); // @output {"m":{},"l":[],"x":1}
    io:println(C is readonly); // @output true
}
