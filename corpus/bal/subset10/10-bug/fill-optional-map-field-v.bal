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

type Outer record {
    map<int> m?;
};

type ClosedOuter record {|
    map<int> m?;
    int z?;
|};

type RestOuter record {|
    int a?;
    map<int>...;
|};

type Inner record {|
    int y?;
|};

public function main() {
    Outer o = {};
    o.m["k"] = 2;
    o["m"]["j"] = 3;
    io:println(o); // @output {"m":{"k":2,"j":3}}

    ClosedOuter c = {};
    c.m["k"] = 4;
    io:println(c); // @output {"m":{"k":4}}

    RestOuter ro = {};
    ro["x"]["k"] = 1;
    io:println(ro); // @output {"x":{"k":1}}

    map<Inner> inner = {};
    map<Inner|map<string>> wide = inner;
    wide["k"]["y"] = 5;
    io:println(inner); // @output {"k":{"y":5}}
}
