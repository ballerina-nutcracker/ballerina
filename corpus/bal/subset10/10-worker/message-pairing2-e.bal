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


function cyclicReceive() {
    worker a {
        int x = <- b; // @error worker message deadlock
        x -> b;
    }
    worker b {
        int y = <- a;
        y -> a;
    }
}

function waitDeadlock() {
    worker a {
        wait b;
        1 -> function;
    }
    worker b {
        int _ = <- function;
    }
    int x = <- a; // @error a waits for b, which waits for the default worker
    x -> b;
}

function deliveryDeadlock() {
    worker a {
        1 -> b;
    }
    worker b {
        int _ = <- function;
        int _ = <- a;
    }
    wait a; // @error a can't terminate before b receives its message
    2 -> b;
}

function severalFaults() {
    worker a {
        1 -> function;
        2 -> function;
    }
    worker b {
        int _ = <- function; // @error the first fault in pairing order is reported
    }
    int _ = <- a;
}

function countBeforeDeadlock() {
    worker a {
        int _ = <- b; // @error no matching send
        2 -> b;
    }
    worker b {
        int y = <- a;
        var _ = y;
    }
}

function multipleReceiveDeadlock() {
    worker a {
        int _ = <- function;
        1 -> function;
    }
    worker b {
        2 -> function;
    }
    record {} _ = <- {a, b}; // @error a waits for the default worker first
    3 -> a;
}

function syncSendDeadlock() {
    worker a {
        1 ->> b;
    }
    worker b {
        int _ = <- function;
        int _ = <- a;
    }
    wait a; // @error a's sync send waits for b, which waits for the default worker
    2 -> b;
}

function flushDeadlock() {
    worker a {
        1 -> b;
        flush b;
        2 -> function;
    }
    worker b {
        int _ = <- function;
        int _ = <- a;
    }
    int _ = <- a; // @error a's flush waits for b, which waits for the default worker
    3 -> b;
}

function bareFlushDeadlock() {
    worker a {
        1 -> b;
        flush;
        2 -> function;
    }
    worker b {
        int _ = <- function;
        int _ = <- a;
    }
    int _ = <- a; // @error a's flush waits for b, which waits for the default worker
    3 -> b;
}

function waitCycleDeadlock() {
    worker a {
        wait b;
        int _ = <- function;
    }
    worker b {
        wait a;
    }
    1 -> a; // @error a and b wait for each other, so the default worker can't deliver
}
