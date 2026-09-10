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

final Point & readonly modulePoint = {x: 1, y: 2};
map<int> moduleSpread = {...modulePoint, z: 3};

type Defaulted record {|
    Point base = {...defaultPoint()};
    P3 full = {...defaultPoint(), z: 4};
|};

class Holder {
    P3 initial = {...defaultPoint(), z: 5};
    P3 assigned;

    function init(P3 assigned) {
        self.assigned = assigned;
    }
}

isolated function defaultPoint() returns Point => {x: 6, y: 7};

function total(P3 p) returns int => p.x + p.y + p.z;

function named(int factor, P3 p) returns int => factor * p.z;

function returnsSpread(Point p) returns P3 {
    return {...p, z: 8};
}

function returnsSpreadExpr(Point p) returns P3 => {...p, z: 9};

public function main() {
    Point p = {x: 1, y: 2};

    io:println(moduleSpread); // @output {"x":1,"y":2,"z":3}

    io:println(total({...p, z: 10})); // @output 13

    io:println(named(factor = 2, p = {...p, z: 11})); // @output 22

    io:println(named(3, {...p, z: 12})); // @output 36

    io:println(returnsSpread(p)); // @output {"x":1,"y":2,"z":8}

    io:println(returnsSpreadExpr(p)); // @output {"x":1,"y":2,"z":9}

    Defaulted defaulted = {};
    io:println(defaulted); // @output {"base":{"x":6,"y":7},"full":{"x":6,"y":7,"z":4}}

    Holder holder = new ({...p, z: 13});
    io:println(holder.initial); // @output {"x":6,"y":7,"z":5}
    io:println(holder.assigned); // @output {"x":1,"y":2,"z":13}

    P3[] list = [{...p, z: 14}, {...p, z: 15}];
    io:println(list); // @output [{"x":1,"y":2,"z":14},{"x":1,"y":2,"z":15}]

    map<P3> nested = {a: {...p, z: 16}};
    io:println(nested); // @output {"a":{"x":1,"y":2,"z":16}}

    io:println(total({...p, z: 17}) + total({...p, z: 18})); // @output 41
}
