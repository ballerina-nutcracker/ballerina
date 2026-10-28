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


function undefinedPeer() {
    worker a {
        1 -> nobody; // @error undefined worker
    }
}

function selfSend() {
    worker a {
    }
    1 -> function; // @error the default worker can't send to itself
}

function selfReceive() {
    worker a {
    }
    int _ = <- function; // @error the default worker can't receive from itself
}

function initStatements() {
    int _ = <- w; // @error workers aren't in scope before their declarations
    1 -> function; // @error the initialization statements belong to the default worker
    worker w {
    }
}

function noWorkers() {
    1 -> function; // @error a body without workers still has a default worker
}

function duplicatePeer() {
    worker a {
        1 -> function;
        2 -> function;
    }
    record {} _ = <- {first: a, second: a}; // @error a peer appears at most once
}

function namedSelfSend() {
    worker a {
        1 -> a; // @error a worker can't send to itself
    }
}

function multipleReceiveUndefinedPeer() {
    worker a {
        1 -> function;
    }
    record {} _ = <- {a, nobody}; // @error undefined worker
}

function duplicateField() {
    worker a {
        1 -> function;
    }
    worker b {
        2 -> function;
    }
    record {} _ = <- {x: a, x: b}; // @error field names are distinct
}

function rejectedWorker(int a) { // @error the parameter is unused
    worker a { // @error a parameter already has the name
        wait b;
        1 -> function;
    }
    worker b {
    }
}
