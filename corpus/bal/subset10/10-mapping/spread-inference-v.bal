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

type Point record {|
    int x;
    int y;
|};

type OptionalY record {|
    int x;
    int y?;
|};

type OpenPoint record {|
    int x;
    string...;
|};

type Inner record {|
    int v;
|};

public function main() {
    Point p = {x: 1, y: 2};
    var required = {...p};
    io:println(required.x + required.y); // @output 3
    io:println(required is record {| int x; int y; |}); // @output true

    OptionalY optional = {x: 1};
    var inferred = {...optional};
    io:println(inferred is record {| int x; int y?; |}); // @output true
    io:println(inferred is record {| int x; int y; |}); // @output false
    io:println(inferred.length()); // @output 1

    OpenPoint openSrc = {x: 1, "tag": "t"};
    var openInferred = {...openSrc};
    io:println(openInferred is record {| int x; string...; |}); // @output true
    io:println(openInferred is map<int>); // @output false
    io:println(openInferred); // @output {"x":1,"tag":"t"}

    Point|OptionalY united = optional;
    var unionInferred = {...united};
    io:println(unionInferred is record {| int x; int y?; |}); // @output true

    Inner shared = {v: 1};
    var withNested = {inner: shared, ...p};
    io:println(withNested.inner.v); // @output 1
    io:println(withNested is record {| Inner inner; int x; int y; |}); // @output true

    var extended = {...p, z: 3};
    io:println(extended is record {| int x; int y; int z; |}); // @output true
    map<int> widened = extended;
    widened["x"] = 10;
    io:println(widened); // @output {"x":10,"y":2,"z":3}

    OpenPoint restSrc = {x: 1};
    var restInferred = {...restSrc};
    map<int|string> restWidened = restInferred;
    restWidened["tag"] = "t";
    io:println(restWidened); // @output {"x":1,"tag":"t"}
}
