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

function branches(boolean flag) {
    worker w returns int {
        return 1;
    }
    if flag {
        int first = wait w;
        io:println(first);
    } else {
        int second = wait w; // @error duplicate reference in the other arm
        io:println(second);
    }
}

function sequential() {
    worker w returns int {
        return 1;
    }
    int first = wait w;
    int second = wait w; // @error duplicate sequential reference
    io:println(first + second);
}

function conditionalExpr(boolean flag) {
    worker w returns int {
        return 1;
    }
    future<int> chosen = flag ? w : w; // @error duplicate reference in conditional arms
    io:println(chosen);
}

function repeatedInWait() {
    worker w returns int {
        return 1;
    }
    var both = wait {first: w, second: w}; // @error repeated within one multiple wait
    io:println(both);
}
