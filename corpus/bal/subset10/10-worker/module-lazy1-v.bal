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

// Workers a and b of f both reach x, whose initializer is still being resolved
// by one of them while it waits on the worker of x's own lambda.
var f = function () returns int {
    worker a returns int {
        return x;
    }
    worker b returns int {
        return x;
    }
    int ra = wait a;
    int rb = wait b;
    return ra + rb + x;
};

var x = call(function () returns int {
    worker w returns int {
        return 1;
    }
    int r = wait w;
    return r;
});

function call(function () returns int fn) returns int {
    return fn();
}

public function main() {
    io:println(f()); // @output 3
}
