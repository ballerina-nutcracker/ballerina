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

const annotation Marker on source var;

const annotation Meta Info on source var;

const annotation Meta[] Infos on source var;

class Counter {
    function next() returns int {
        @Marker
        int n = 7;
        return n;
    }
}

public function main() {
    @Marker
    int w = 2;
    io:println(w); // @output 2

    @Marker
    int noInit;
    noInit = 3;
    io:println(noInit); // @output 3

    @Marker
    final int f = 4;
    io:println(f); // @output 4

    @Marker
    var inferred = 5;
    io:println(inferred); // @output 5

    @Info {name: "literal"}
    string s = "a";
    io:println(s); // @output a

    @Info {name: NAME}
    string c = "b";
    io:println(c); // @output b

    @Infos {name: "first"}
    @Infos {name: "second"}
    string r = "c";
    io:println(r); // @output c

    @Marker
    @Info {name: "both"}
    string m = "d";
    io:println(m); // @output d

    if w == 2 {
        @Marker
        int nested = 6;
        io:println(nested); // @output 6
    }

    io:println(new Counter().next()); // @output 7

    var fn = function() returns int {
        @Marker
        int inLambda = 8;
        return inLambda;
    };
    io:println(fn()); // @output 8
}
