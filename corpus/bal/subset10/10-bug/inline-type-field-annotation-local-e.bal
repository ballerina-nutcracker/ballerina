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

class Point {
    int x = 1;
}

public function main() {
    record {| @OnType int x; |} wrongPoint = {x: 1}; // @error
    io:println(wrongPoint.x);

    object { @RecordFieldMarker int x; } wrongObjectPoint = new Point(); // @error
    io:println(wrongObjectPoint.x);

    record {| @RecordFieldMarker @RecordFieldMarker int x; |} duplicate = {x: 2}; // @error
    io:println(duplicate.x);

    record {| @RecordFieldMarker {} int x; |} markerWithValue = {x: 3}; // @error
    io:println(markerWithValue.x);

    string local = "x";
    record {| @SourceFieldInfo {name: local} int x; |} notConstant = {x: 4}; // @error
    io:println(notConstant.x);

    map<record {| @OnType int x; |}> nestedInMap = {}; // @error
    io:println(nestedInMap.length());
}
