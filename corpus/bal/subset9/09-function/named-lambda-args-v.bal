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

type Pair record {|
    int left;
    int right;
|};

type F function (int a = 5) returns int;

public function main() {
    function(int a = 5) returns int base = function(int x = 10) returns int {
        return x;
    };
    io:println(base()); // @output 5
    io:println(base(a = 6)); // @output 6

    function(int first, int second = first + 3) returns int localFn = function(int x, int y = 100) returns int {
        return x + y;
    };
    io:println(localFn(4)); // @output 11
    io:println(localFn(first = 4, second = 5)); // @output 9

    function(int z = 7) returns int copied = base;
    io:println(copied()); // @output 7

    io:println(takesFn(base)); // @output 20

    function(*Pair p) returns int recordFn = function(Pair q) returns int {
        return q.left + q.right;
    };
    io:println(recordFn(left = 8, right = 9)); // @output 17

    io:println(applyFns(10, function(int b = 10) returns int {
        return b;
    }, function(int c = 6) returns int {
        return c + 1;
    })); // @output 11

    io:println(applyInlineFns(10, function(int b = 10) returns int {
        return b;
    }, function(int c = 6) returns int {
        return c + 1;
    })); // @output 11
}

function takesFn(function(int g = 20) returns int f) returns int {
    return f();
}

function applyFns(int a, F... fns) returns int {
    if fns.length() == 0 {
        return a;
    }
    int res = a;
    foreach F fn in fns {
        res = fn(res);
    }
    return res;
}

function applyInlineFns(int a, function(int a = 5) returns int... fns) returns int {
    if fns.length() == 0 {
        return a;
    }
    int res = a;
    foreach function(int a = 5) returns int fn in fns {
        res = fn(res);
    }
    return res;
}
