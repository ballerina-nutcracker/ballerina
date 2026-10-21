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

type A record {|
    boolean x;
|};

type B record {|
    string x;
|};

type Run object {
    function run(int p) returns int;
};

type C record {|
    Run x;
    int m;
|};

type D record {|
    Run x;
    string m;
|};

class Runner {
    function run(int p = 0) returns int {
        return p + 1;
    }
}

public function main() {
    any v = new Runner();
    A|B r = {x: v is object { function run(int p = 3) returns int; }};
    C|D c = {x: <object { function run(int p = 3) returns int; }>v, m: 1};
    io:println(r); // @output {"x":true}
    io:println(c is C); // @output true
    io:println(c.x.run(4)); // @output 5
    if v is object { function run(int p = 3) returns int; } {
        io:println(v.run(4)); // @output 5
    }
}
