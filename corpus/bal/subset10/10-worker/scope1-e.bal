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

function workerNameInInitRegion() {
    future<int> unavailable = w; // @error worker names are unavailable in the initialization region
    worker w returns int {
        return 1;
    }
    io:println(unavailable);
}

function selfReference() {
    worker w returns int {
        int own = wait w; // @error a worker is not visible inside its own body
        return own;
    }
    int result = wait w;
    io:println(result);
}

function postWorkerLocal() {
    worker w returns int {
        return later; // @error later is declared after the worker declarations
    }
    int later = 1;
    int result = wait w;
    io:println(later + result);
}
