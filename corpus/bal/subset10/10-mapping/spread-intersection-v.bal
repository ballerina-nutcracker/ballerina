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

type IntOptX record {|
    int x?;
|};

type StringOptX record {|
    string x?;
|};

type OpenInt record {|
    int...;
|};

type HasY record {|
    int y;
    int...;
|};

public function main() {
    IntOptX & StringOptX noX = {};
    var a = {x: 1, ...noX};
    io:println(a); // @output {"x":1}
    io:println(a is record {| int x; |}); // @output true

    OpenInt & HasY restProvided = {y: 2};
    var b = {...restProvided};
    io:println(b); // @output {"y":2}
    io:println(b is record {| int y; int...; |}); // @output true

    IntOptX intOnly = {x: 3};
    var c = {...intOnly};
    io:println(c is record {| int x?; |}); // @output true
    io:println(c); // @output {"x":3}
}
