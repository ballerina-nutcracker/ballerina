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

annotation ParamMarker on parameter;

annotation ReturnMarker on return;

const annotation Meta SourceParamInfo on source parameter;

public function main() {
    var paramPoint = function(@OnType int x) returns int => x; // @error
    io:println(paramPoint(1));

    var restPoint = function(@OnType int... xs) returns int => xs.length(); // @error
    io:println(restPoint(1));

    var returnPoint = function() returns @OnType int => 1; // @error
    io:println(returnPoint());

    var swapped = function(@ReturnMarker int x) returns int => x; // @error
    io:println(swapped(1));

    var duplicate = function(@ParamMarker @ParamMarker int x) returns int => x; // @error
    io:println(duplicate(1));

    var markerWithValue = function(@ParamMarker {} int x) returns int => x; // @error
    io:println(markerWithValue(1));

    string local = "x";
    var notConstant = function(@SourceParamInfo {name: local} int x) returns int => x; // @error
    io:println(notConstant(1));
}
