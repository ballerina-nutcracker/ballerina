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

type Shift record {|
    int dx = 0;
    int dy = 0;
|};

type Point record {| int x; int y; |};

public function main() {
    Point p = {x: 1, y: 2};

    Point q = shift(p, dx = 10, dy = 20);
    io:println(q.x); // @output 11
    io:println(q.y); // @output 22

    Point r = shift(p, {dx: 5});
    io:println(r.x); // @output 6
    io:println(r.y); // @output 2

    Point s = shift(p);
    io:println(s.x); // @output 1
    io:println(s.y); // @output 2
}

function shift(Point p, *Shift opts, typedesc retTy = <>) returns retTy = external;
