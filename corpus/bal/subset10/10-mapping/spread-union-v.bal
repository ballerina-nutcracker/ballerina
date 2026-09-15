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

type Both record {|
    int x;
    int y;
|};

type OnlyX record {|
    int x;
|};

type StringX record {|
    string x;
|};

public function main() {
    Both both = {x: 1, y: 2};
    Both|OnlyX either = both;
    var inferred = {...either};
    io:println(inferred.x); // @output 1
    io:println(inferred is record {| int x; int y?; |}); // @output true
    io:println(inferred is record {| int x; int y; |}); // @output false

    OnlyX onlyX = {x: 3};
    OnlyX|StringX widened = onlyX;
    var unionValues = {...widened};
    io:println(unionValues.x); // @output 3
    io:println(unionValues is record {| int|string x; |}); // @output true

    map<int> target = {...onlyX, y: 5};
    io:println(target); // @output {"x":3,"y":5}
}
