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

// @productions list-type-descriptor list-constructor-expr bitwise-and-expr bitwise-xor-expr bitwise-or-expr exact-equality equality multiplicative-expr equality-expr relational-expr return-stmt additive-expr function-call-expr local-var-decl-stmt int-literal
import ballerina/io;

function one() returns int {
    io:println(1);
    return 1;
}

function two() returns int {
    io:println(2);
    return 2;
}

public function main() {
    int mul = one() * two(); // @output 1
    // @output 2
    _ = mul;

    int div = one() / two(); // @output 1
    // @output 2
    _ = div;

    int rmd = one() % two(); // @output 1
    // @output 2
    _ = rmd;

    int add = one() + two(); // @output 1
    // @output 2
    _ = add;

    int sub = one() - two(); // @output 1
    // @output 2
    _ = sub;

    boolean lt = one() < two(); // @output 1
    // @output 2
    _ = lt;

    boolean lteq = one() <= two(); // @output 1
    // @output 2
    _ = lteq;

    boolean gt = one() > two(); // @output 1
    // @output 2
    _ = gt;

    boolean gteq = one() >= two(); // @output 1
    // @output 2
    _ = gteq;

    boolean eq = one() == two(); // @output 1
    // @output 2
    _ = eq;

    boolean neq = one() != two(); // @output 1
    // @output 2
    _ = neq;

    boolean eeq = one() === two(); // @output 1
    // @output 2
    _ = eeq;

    boolean neeq = one() !== two(); // @output 1
    // @output 2
    _ = neeq;

    int and = one() & two(); // @output 1
    // @output 2
    _ = and;

    int xor = one() ^ two(); // @output 1
    // @output 2
    _ = xor;

    int or = one() | two(); // @output 1
    // @output 2
    _ = or;

    any[] arr = [one(), two()]; // @output 1
    // @output 2
    _ = arr;

    ignore(one(), two()); // @output 1
    // @output 2
}

function ignore(int a, int b) {
    int _ = a;
    int _ = b;
}
