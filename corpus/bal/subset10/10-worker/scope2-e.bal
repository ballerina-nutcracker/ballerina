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

// A worker name shares the enclosing scope, so it cannot shadow a local.
function collidesWithLocal() {
    int w = 1;
    worker w returns int { // @error Variable already defined: w
        return 2;
    }
    io:println(w);
}

// Two workers in one function share the same scope, so their names collide.
function collidesWithSiblingWorker() {
    worker w returns int {
        return 1;
    }
    worker w returns int { // @error Variable already defined: w
        return 2;
    }
}

// A worker name collides with a parameter of the declaring function.
function collidesWithParameter(int w) {
    worker w returns int { // @error Variable already defined: w
        return 1;
    }
    io:println(w);
}

// The worker region precedes the trailing statements, so a local declared
// after it collides with the worker name too.
function collidesWithLaterLocal() {
    worker w returns int {
        return 1;
    }
    int w = 2; // @error Variable already defined: w
    io:println(w);
}

// A worker body is nested in the declaring function, so a worker local cannot
// shadow a local of that function.
function localShadowsEnclosingLocal() {
    int x = 1;
    worker w returns int {
        int x = 2; // @error Variable already defined: x
        return x;
    }
    int result = wait w;
    io:println(x + result);
}
