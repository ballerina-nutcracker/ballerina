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

type Point record {
    int x;
    int y;
};

type ColouredPoint record {
    int x;
    int y;
    string colour;
};

type MovePointFn function (Point, int, int) returns Point;

type MoveColouredPointFn function (ColouredPoint, int, int) returns ColouredPoint;

type MoveFn MovePointFn & MoveColouredPointFn;

public function main() {
    MoveFn fn = moveFn;
    ColouredPoint cp = {x: 10, y: 20, colour: "red"};
    Point cp2 = fn(cp, 10, 20);
    io:println(cp2); // @output {"x":20,"y":40,"colour":"red"}
    Point p = {x: 10, y: 20};
    Point p2 = fn(p, 10, 20);
    io:println(p2); // @output {"x":20,"y":40,"colour":"white"}
}

function moveFn(Point p, int dx, int dy) returns ColouredPoint {
    if p is ColouredPoint {
        return moveColoredPoint(p, dx, dy);
    }
    Point movedPoint = movePoint(p, dx, dy);
    return {x: movedPoint.x, y: movedPoint.y, colour: "white"};
}

function movePoint(Point p, int dx, int dy) returns Point {
    p.x = p.x + dx;
    p.y = p.y + dy;
    return p;
}

function moveColoredPoint(ColouredPoint p, int dx, int dy) returns ColouredPoint {
    if p.colour != "white" {
        p.x = p.x + dx;
        p.y = p.y + dy;
    }
    return p;
}
