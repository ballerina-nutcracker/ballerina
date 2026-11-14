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

const NAME = "fromConst";

annotation ParamMarker on parameter;

annotation ReturnMarker on return;

annotation Meta ParamInfo on parameter;

annotation Meta[] ParamInfos on parameter;

const annotation Meta SourceParamInfo on source parameter;

type Holder record {|
    function (int) returns int fn = function(@ParamMarker int x) returns @ReturnMarker int => x + 1;
|};

function (int) returns int moduleFn = function(@ParamMarker int x) returns @ReturnMarker int => x + 2;

function apply(function (int) returns int fn = function(@ParamInfo {name: LATE} int x) returns int => x + 3) returns int {
    return fn(1);
}

const LATE = "declaredLater";

public function main() {
    var required = function(@ParamMarker int x) returns int {
        return x;
    };
    io:println(required(1)); // @output 1

    var defaultable = function(@ParamMarker int x = 2) returns int {
        return x;
    };
    io:println(defaultable()); // @output 2

    var rest = function(@ParamMarker int... xs) returns int {
        return xs.length();
    };
    io:println(rest(1, 2, 3)); // @output 3

    var returnAnnotated = function() returns @ReturnMarker int {
        return 4;
    };
    io:println(returnAnnotated()); // @output 4

    var exprBody = function(@ParamMarker int x) returns @ReturnMarker int => x;
    io:println(exprBody(5)); // @output 5

    var literalValue = function(@ParamInfo {name: "literal"} int x) returns int => x;
    io:println(literalValue(6)); // @output 6

    var constValue = function(@SourceParamInfo {name: NAME} int x) returns int => x;
    io:println(constValue(7)); // @output 7

    string local = "local";
    var runtimeValue = function(@ParamInfo {name: local} int x) returns int => x;
    io:println(runtimeValue(8)); // @output 8

    var repeated = function(@ParamInfos {name: "a"} @ParamInfos {name: "b"} int x) returns int => x;
    io:println(repeated(9)); // @output 9

    Holder holder = {};
    var held = holder.fn;
    io:println(held(9)); // @output 10

    io:println(moduleFn(9)); // @output 11

    io:println(apply()); // @output 4
}
