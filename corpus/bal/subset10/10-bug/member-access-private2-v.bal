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

class A {
    private int x;
    private int y = 10;
    function (A) returns int getter = function(A o) returns int {
        return o.x;
    };

    function init(int x) {
        self.x = x;
    }

    private function double() returns int {
        return self.x * 2;
    }

    function nested() returns int {
        var f = function() returns int {
            var g = function() returns int {
                return self.double() + self.y;
            };
            return g();
        };
        return f();
    }

    function bump(A other) {
        other.x += 1;
        other.y -= 1;
    }

    function sumWhile(A[] xs) returns int {
        int i = 0;
        int total = 0;
        while i < xs.length() {
            total += xs[i].x;
            i += 1;
        }
        return total;
    }

    function arrow() returns int {
        function (A) returns int f = o => o.x;
        return f(self);
    }

    function query(A[] xs) returns int[] {
        return from A o in xs select o.y;
    }

    function narrowed(A|int v) returns int {
        if v is A {
            return v.double();
        }
        return v;
    }
}

client class R {
    private int v = 5;

    resource function get val() returns int {
        return self.v + self.priv();
    }

    remote function rem() returns int {
        return self.v * self.priv();
    }

    private function priv() returns int {
        return 1;
    }
}

class M {
    int m = 1;
}

function readModuleMember(M m) returns int {
    m.m += 2;
    return m.m;
}

public function main() returns error? {
    A a = new (1);
    A b = new (5);
    io:println(a.nested()); // @output 12
    a.bump(b);
    io:println(a.sumWhile([a, b])); // @output 7
    io:println(b.arrow()); // @output 6
    io:println(a.query([a, b])); // @output [10,9]
    io:println(a.narrowed(b)); // @output 12
    function (A) returns int g = a.getter;
    io:println(g(b)); // @output 6
    R r = new;
    int v = r->/val;
    io:println(v); // @output 6
    int w = r->rem();
    io:println(w); // @output 5
    io:println(readModuleMember(new M())); // @output 3
}
