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

type R record {|
    int '_;
    int x;
|};

type Rest record {|
    int x;
    int...;
|};

function f(*R r) {
    io:println(r);
}

function h(*Rest r) {
    io:println(r);
}

function g(int '_, int y = 2) {
    io:println('_ + y);
}

class C {
    function init(*R r) {
        io:println(r);
    }
}

public function main() {
    f('_ = 1, x = 2); // @output {"_":1,"x":2}
    _ = new C('_ = 7, x = 8); // @output {"_":7,"x":8}
    h(x = 3, '_ = 4); // @output {"x":3,"_":4}
    g('_ = 1); // @output 3
}
