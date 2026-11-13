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

type P3 record {|
    int x;
    int y;
    int z;
|};

int defaultEvaluations = 0;

function nextTag() returns int {
    defaultEvaluations += 1;
    return defaultEvaluations;
}

isolated function basePoint() returns Point => {x: 1, y: 2};

// The default spreads the preceding parameter.
function fromPrevious(Point p, P3 q = {...p, z: 10}) returns P3 => q;

// Each default spreads the parameter declared just before it.
function chained(Point p, P3 q = {...p, z: 20}, map<int> r = {...q, w: 30}) returns map<int> => r;

// The default spreads a call, so it is re-evaluated per call.
function fromCall(P3 p = {...basePoint(), z: nextTag()}) returns P3 => p;

// The default value must be a fresh mapping on every call.
function fresh(Point p, map<int> m = {...p}) returns map<int> {
    m["size"] = m.length();
    return m;
}

// A defaulted parameter of an object's init method.
class Box {
    P3 value;

    function init(Point p, P3 q = {...p, z: 40}) {
        self.value = q;
    }
}

public function main() {
    Point p = {x: 1, y: 2};

    io:println(fromPrevious(p)); // @output {"x":1,"y":2,"z":10}
    io:println(fromPrevious(p, {x: 3, y: 4, z: 5})); // @output {"x":3,"y":4,"z":5}

    io:println(chained(p)); // @output {"x":1,"y":2,"z":20,"w":30}
    io:println(chained(p, {x: 6, y: 7, z: 8})); // @output {"x":6,"y":7,"z":8,"w":30}
    io:println(chained(p, r = {a: 9})); // @output {"a":9}

    io:println(fromCall()); // @output {"x":1,"y":2,"z":1}
    io:println(fromCall()); // @output {"x":1,"y":2,"z":2}
    io:println(fromCall({x: 0, y: 0, z: 0})); // @output {"x":0,"y":0,"z":0}
    io:println(defaultEvaluations); // @output 2

    io:println(fresh(p)); // @output {"x":1,"y":2,"size":2}
    io:println(fresh(p)); // @output {"x":1,"y":2,"size":2}

    Box defaulted = new (p);
    io:println(defaulted.value); // @output {"x":1,"y":2,"z":40}

    Box supplied = new (p, {x: 11, y: 12, z: 13});
    io:println(supplied.value); // @output {"x":11,"y":12,"z":13}
}
