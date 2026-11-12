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

annotation RecordFieldMarker on record field;

annotation ObjectFieldMarker on object field;

annotation FieldMarker on field;

annotation Meta FieldInfo on record field;

const annotation Meta SourceFieldInfo on source record field;

record {| @RecordFieldMarker int x; |} moduleVar = {x: 1};

type Nested record {|
    record {| @RecordFieldMarker int y; |} inner;
|};

type RecordArray record {| @RecordFieldMarker int x; |}[];

type RecordOrInt record {| @RecordFieldMarker int x; |}|int;

type ReadonlyRecord readonly & record {| @RecordFieldMarker int x; |};

type WithObject record {|
    object { @ObjectFieldMarker int x; } obj;
|};

type LateConst record {|
    record {| @SourceFieldInfo {name: LATE} int y; |} inner;
|};

const LATE = "declaredLater";

class Point {
    int x = 9;
}

class Holder {
    record {| @RecordFieldMarker int x; |} held = {x: 10};
}

function sum(record {| @RecordFieldMarker int x; |} a, record {| @FieldInfo {name: "b"} int x; |} b) returns int {
    return a.x + b.x;
}

function make() returns record {| @RecordFieldMarker int x; |} {
    return {x: 12};
}

public function main() {
    io:println(moduleVar.x); // @output 1

    Nested nested = {inner: {y: 2}};
    io:println(nested.inner.y); // @output 2

    RecordArray array = [{x: 3}];
    io:println(array[0].x); // @output 3

    RecordOrInt recordOrInt = 4;
    io:println(recordOrInt); // @output 4

    ReadonlyRecord readonlyRecord = {x: 5};
    io:println(readonlyRecord.x); // @output 5

    record {| @RecordFieldMarker int x; |} local = {x: 6};
    io:println(local.x); // @output 6

    record {| @FieldMarker int x; |} generic = {x: 7};
    io:println(generic.x); // @output 7

    record {| @FieldInfo {name: "literal"} @RecordFieldMarker int x; |} valued = {x: 8};
    io:println(valued.x); // @output 8

    object { @ObjectFieldMarker @FieldMarker int x; } obj = new Point();
    io:println(obj.x); // @output 9

    io:println(new Holder().held.x); // @output 10

    io:println(sum({x: 5}, {x: 6})); // @output 11

    io:println(make().x); // @output 12

    WithObject withObject = {obj: new Point()};
    io:println(withObject.obj.x + 4); // @output 13

    LateConst lateConst = {inner: {y: 14}};
    io:println(lateConst.inner.y); // @output 14

    map<record {| @RecordFieldMarker int x; |}> mapped = {a: {x: 15}};
    io:println(mapped["a"]?.x); // @output 15

    string name = "local";
    record {| @FieldInfo {name: name} int x; |} runtimeValue = {x: 16};
    io:println(runtimeValue.x); // @output 16
}
