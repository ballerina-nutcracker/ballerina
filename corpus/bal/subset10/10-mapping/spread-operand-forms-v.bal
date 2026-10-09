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

class Box {
    Point p;

    function init(Point p) {
        self.p = p;
    }

    function get() returns Point => self.p;
}

function makePoint() returns Point => {x: 1, y: 2};

function mayFail(boolean bad) returns Point|error =>
        bad ? error("bad operand") : {x: 3, y: 4};

function optional(boolean present) returns Point? => present ? {x: 5, y: 6} : ();

public function main() {
    var fromCall = {...makePoint(), z: 1};
    io:println(fromCall); // @output {"x":1,"y":2,"z":1}

    Box box = new ({x: 7, y: 8});
    var fromMethod = {...box.get(), z: 2};
    io:println(fromMethod); // @output {"x":7,"y":8,"z":2}

    var fromField = {...box.p, z: 3};
    io:println(fromField); // @output {"x":7,"y":8,"z":3}

    map<Point> holder = {a: {x: 9, y: 10}};
    var fromMember = {...holder.get("a"), z: 4};
    io:println(fromMember); // @output {"x":9,"y":10,"z":4}

    var fromCast = {...<Point>optional(true), z: 6};
    io:println(fromCast); // @output {"x":5,"y":6,"z":6}

    var fromConditional = {...(true ? {x: 11, y: 12} : {x: 13, y: 14}), z: 7};
    io:println(fromConditional); // @output {"x":11,"y":12,"z":7}

    Point? narrowed = optional(true);
    if narrowed is Point {
        var fromNarrowed = {...narrowed, z: 8};
        io:println(fromNarrowed); // @output {"x":5,"y":6,"z":8}
    }

    var fromCheckpanic = {...checkpanic mayFail(false), z: 9};
    io:println(fromCheckpanic); // @output {"x":3,"y":4,"z":9}

    var fromNested = {...{...makePoint(), z: 10}, w: 11};
    io:println(fromNested); // @output {"x":1,"y":2,"z":10,"w":11}
}
