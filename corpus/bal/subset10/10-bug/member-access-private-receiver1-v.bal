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

readonly class RO {
    private int x;

    function init(int x) {
        self.x = x;
    }

    private function dbl() returns int {
        return self.x * 2;
    }

    function peer(RO o) returns int {
        return o.x + o.dbl();
    }
}

class A {
    private int x;

    function init(int x) {
        self.x = x;
    }

    private function dbl() returns int {
        return self.x * 2;
    }

    function ro(readonly & A o) returns int {
        return o.x + o.dbl();
    }

    function nil(A? o) returns int {
        if o is () {
            return -1;
        }
        return o.x + o.dbl();
    }

    function viaAlias(TA o) returns int {
        return o.x;
    }

    function mixed(A|M o) returns int {
        return o.x;
    }
}

type TA A;

distinct class D {
    private int x = 3;

    private function get() returns int {
        return self.x;
    }

    function peer(D o) returns int {
        return o.x + o.get();
    }
}

isolated class I {
    private int x = 4;

    isolated function get() returns int {
        lock {
            return self.x;
        }
    }
}

class M {
    int x = 8;
}

public function main() {
    A a = new (1);
    io:println(new RO(2).peer(new RO(3))); // @output 9
    io:println(a.nil(a)); // @output 3
    io:println(a.nil(())); // @output -1
    io:println(a.viaAlias(a)); // @output 1
    io:println(a.mixed(new M())); // @output 8
    io:println(new D().peer(new D())); // @output 6
    io:println(new I().get()); // @output 4
}
