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

readonly class RO {
    private int x;

    function init(int x) {
        self.x = x;
    }

    private function dbl() returns int {
        return self.x * 2;
    }
}

class A {
    private int x = 1;

    private function dbl() returns int {
        return self.x * 2;
    }
}

distinct class D {
    private int x = 3;
}

isolated class I {
    private final int x = 4;
}

type TA A;

function f1(RO r) returns int {
    return r.x; // @error private field of a readonly class
}

function f2(RO r) returns int {
    return r.dbl(); // @error private method of a readonly class
}

function f3(readonly & A a) returns int {
    return a.x; // @error private field via a readonly intersection
}

function f4(A & readonly a) returns int {
    return a.dbl(); // @error private method via a readonly intersection
}

function f5(D d) returns int {
    return d.x; // @error private field of a distinct class
}

function f6(I i) returns int {
    return i.x; // @error private field of an isolated class
}

function f7(TA a) returns int {
    return a.x; // @error private field via a type alias
}

function f8(A? a) returns int {
    if a == () {
        return 0;
    }
    return a.x; // @error private field after nil narrowing
}

public function main() {
}
