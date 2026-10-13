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

class A {
    private int x = 1;

    private function f() returns int {
        return 1;
    }
}

class B {
    int y = (new A()).x; // @error private field is not visible in a field initializer of another class
    function (A) returns int getter = function(A o) returns int {
        return o.x; // @error private field is not visible in a lambda in a field initializer of another class
    };

    private int x = 2;

    function viaUnion(A|B v) returns int {
        return v.x; // @error private field of A is not visible in methods of B
    }
}

client class R {
    resource function get val(A a) returns int {
        return a.x; // @error private field is not visible in a resource method of another class
    }

    remote function rem(A a) returns int {
        return a.f(); // @error private method is not visible in a remote method of another class
    }
}

A ga = new;

function viaLambda() returns int {
    var g = function() returns int {
        return ga.x; // @error private field is not visible in a lambda outside the class
    };
    return g();
}

function compound() {
    ga.x += 1; // @error private field is not visible in a compound assignment outside the class
}

public function main() {
}
