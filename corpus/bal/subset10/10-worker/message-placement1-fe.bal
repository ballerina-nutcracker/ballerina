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


function inIf(boolean flag) {
    worker a {
        int _ = <- function;
    }
    if flag {
        1 -> a; // @error worker message send/receive is not supported here
    }
}

function inWhile(boolean flag) {
    worker a {
        1 -> function;
    }
    while flag {
        int _ = <- a; // @error worker message send/receive is not supported here
    }
}

function inForeach() {
    worker a {
        int _ = <- function;
    }
    foreach int i in 0 ..< 1 {
        i -> a; // @error worker message send/receive is not supported here
    }
}

function inMatchClause(int value) {
    worker a {
        int _ = <- function;
    }
    match value {
        1 => {
            1 -> a; // @error worker message send/receive is not supported here
        }
    }
}

function inLock() {
    worker a {
        int _ = <- function;
    }
    lock {
        1 -> a; // @error worker message send/receive is not supported here
    }
}

function inBlock() {
    worker a {
        int _ = <- function;
    }
    {
        1 -> a; // @error worker message send/receive is not supported here
    }
}

function throughLambda() {
    worker a {
        int _ = <- function;
    }
    var f = function () {
        1 -> a; // @error worker message send/receive is not supported here
    };
    f();
}

function placementInLambda(boolean flag) {
    worker a {
        var g = function () {
            if flag {
                1 -> b; // @error worker message send/receive is not supported here
            }
        };
        g();
    }
    worker b {
        int _ = <- a;
    }
}
