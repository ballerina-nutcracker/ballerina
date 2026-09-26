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

// Spreading a module level variable inside a default value: a function
// parameter default, an `init` method parameter default, an object field
// default and a record field default. Module level variable types are resolved
// after function signatures, so a reference from a default value has to pull
// the variable's type in lazily; otherwise the operand resolves to `never`.

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

function defaulted(P3 p = {...modulePoint, z: 1}) returns P3 => p;

class Box {
    P3 initial = {...modulePoint, z: 2};
    P3 value;

    function init(P3 p = {...modulePoint, z: 3}) {
        self.value = p;
    }
}

type Defaulted record {|
    Point base = {...modulePoint};
    P3 full = {...modulePoint, z: 4};
|};

public function main() {
    io:println(defaulted()); // @output {"x":1,"y":2,"z":1}
    io:println(defaulted({x: 5, y: 6, z: 7})); // @output {"x":5,"y":6,"z":7}

    Box box = new;
    io:println(box.initial); // @output {"x":1,"y":2,"z":2}
    io:println(box.value); // @output {"x":1,"y":2,"z":3}

    Box supplied = new ({x: 8, y: 9, z: 10});
    io:println(supplied.value); // @output {"x":8,"y":9,"z":10}

    Defaulted fields = {};
    io:println(fields); // @output {"base":{"x":1,"y":2},"full":{"x":1,"y":2,"z":4}}
}
