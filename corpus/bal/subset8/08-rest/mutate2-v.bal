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

type R record {|
    int x;
    boolean y;
    string...;
|};

public function main() {
    R r = {x: 1, y: false};
    map<any> m = r;
    m["stuff"] = "xyzzy";
    m["stuff"] = "abc";
    m["x"] = 2;
    m["y"] = true;
    io:println(r.x); // @output 2
    io:println(r.y); // @output true
    io:println(r["stuff"]); // @output abc
    io:println(m === r); // @output true
}

