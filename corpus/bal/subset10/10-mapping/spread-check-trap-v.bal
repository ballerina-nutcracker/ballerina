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

function produce(string tag, boolean bad) returns Point|error {
    io:println("eval " + tag);
    return bad ? error("bad " + tag) : {x: 1, y: 2};
}

function panicking() returns Point {
    panic error("panicking operand");
}

function checkedSpread(boolean bad) returns P3|error => {...check produce("check", bad), z: 1};

function sink(P3 first, P3 second) returns int => first.z + second.z;

function checkedArgs(boolean bad) returns int|error =>
        sink({...check produce("A", bad), z: 2}, {...check produce("B", false), z: 3});

public function main() {
    io:println(checkedSpread(false));
    // @output eval check
    // @output {"x":1,"y":2,"z":1}

    P3|error failed = checkedSpread(true);
    // @output eval check
    io:println(failed is error); // @output true

    io:println(checkedArgs(false));
    // @output eval A
    // @output eval B
    // @output 5

    int|error failedArgs = checkedArgs(true);
    // @output eval A
    io:println(failedArgs is error); // @output true

    P3|error trapped = trap {...checkpanic produce("trap", true), z: 4};
    // @output eval trap
    io:println(trapped is error); // @output true

    P3|error trappedPanic = trap {...panicking(), z: 5};
    io:println(trappedPanic is error); // @output true

    P3|error trappedOk = trap {...checkpanic produce("ok", false), z: 6};
    // @output eval ok
    io:println(trappedOk); // @output {"x":1,"y":2,"z":6}

    int|error trappedArg = trap sink({...checkpanic produce("arg", true), z: 7}, {x: 0, y: 0, z: 0});
    // @output eval arg
    io:println(trappedArg is error); // @output true
}
