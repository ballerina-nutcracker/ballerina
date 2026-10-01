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


function extraAsyncSend() {
    worker a {
        1 -> function;
        2 -> function; // @error no matching receive
    }
    int _ = <- a;
}

function extraSyncSend() {
    worker a {
        1 ->> function; // @error no matching receive
    }
}

function extraReceive() {
    worker a {
    }
    int _ = <- a; // @error no matching send
}

function extraMultipleReceive() {
    worker a {
        1 -> function;
    }
    worker b {
    }
    record {} _ = <- {a, b}; // @error no matching send from b
}

function receiveAfterWait() {
    worker a {
        1 -> function;
    }
    wait a;
    int _ = <- a; // @error interaction after wait
}

function sendAfterWait() {
    worker a {
        int _ = <- function;
    }
    wait a;
    1 -> a; // @error interaction after wait
}

function sendAfterMultipleWait() {
    worker a {
        int _ = <- function;
    }
    worker b {
    }
    _ = wait {a, b};
    1 -> a; // @error interaction after wait
}

function alternateWaitDeadlock() {
    worker a {
        int _ = <- function;
    }
    worker b {
        int _ = <- function;
    }
    _ = wait a | b; // @error neither worker can finish before the default worker sends
    1 -> a;
    2 -> b;
}
