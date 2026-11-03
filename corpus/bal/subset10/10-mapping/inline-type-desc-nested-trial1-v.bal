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

type C record {|
    int y;
|};

type D record {|
    string y;
|};

public function main() {
    record {| C|D c; [int, int]|[int, string] l; |} value = {c: {y: 1}, l: [1, 2]};
    any v = value;
    A|B r = {x: v is record {| C|D c = {y: 1}; [int, int]|[int, string] l = [1, "s"]; |}};
    A|B p = {x: v is function (C|D c = {y: 1}, [int, int]|[int, string] l = [1, "s"])};
    io:println(r); // @output {"x":true}
    io:println(p); // @output {"x":false}
}
