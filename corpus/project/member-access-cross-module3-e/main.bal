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

import testorg/memberaccesscrossmodule3.foo;

type L object {
    public int a;
};

class C {
    private int x = 10;

    function sameName(foo:C other) returns int {
        return other.x + self.x; // @error private member of foo:C is not visible in a same-named local class
    }
}

function r1(foo:RO r) returns int {
    return r.x; // @error private field of an imported readonly class
}

function r2(foo:RO r) returns int {
    return r.m; // @error module-level field of an imported readonly class
}

function r3(readonly & foo:C c) returns int {
    return c.m; // @error module-level field via a readonly intersection
}

function r4(foo:C? c) returns int {
    if c is () {
        return 0;
    }
    return c.m; // @error module-level field after nil narrowing
}

function i1(foo:PT & L v) returns int {
    return v.b; // @error module-level member of an object intersection
}

public function main() {
}
