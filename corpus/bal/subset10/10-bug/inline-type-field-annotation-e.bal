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

type Meta record {|
    string name;
|};

annotation OnType on type;

annotation RecordFieldMarker on record field;

annotation ObjectFieldMarker on object field;

const annotation Meta SourceFieldInfo on source record field;

record {| @OnType int x; |} moduleVar = {x: 1}; // @error

type Nested record {|
    record {| @OnType int y; |} inner; // @error
|};

type RecordArray record {| @OnType int x; |}[]; // @error

type RecordOrInt record {| @ObjectFieldMarker int x; |}|int; // @error

type WithObject record {|
    object { @RecordFieldMarker int x; } obj; // @error
|};

class Holder {
    record {| @OnType int x; |} held = {x: 2}; // @error
}

function sum(record {| @OnType int x; |} a) returns int { // @error
    return a.x;
}

function make() returns record {| @OnType int x; |} { // @error
    return {x: 3};
}

type Duplicate record {|
    record {|
        @RecordFieldMarker
        @RecordFieldMarker // @error
        int y;
    |} inner;
|};

type MarkerWithValue record {|
    record {| @RecordFieldMarker {} int y; |} inner; // @error
|};

string notConstant = "x";

type NotConstant record {|
    record {| @SourceFieldInfo {name: notConstant} int y; |} inner; // @error
|};

public function main() {
    io:println(moduleVar.x + sum({x: 1}) + make().x + new Holder().held.x);
}
