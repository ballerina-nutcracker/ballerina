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


function incompatibleReceive() {
    worker a {
        int v = 1;
        v -> function;
    }
    string _ = <- a; // @error int can't be assigned to string
}

function unassignedSyncSend() returns error? {
    worker a returns error? {
        int _ = check <- function;
    }
    1 ->> a; // @error a's failure must be handled
    check wait a;
}

function unassignedFlush() returns error? {
    worker a returns error? {
        int _ = check <- function;
    }
    1 -> a;
    flush a; // @error a's failure must be handled
    check wait a;
}

function incompatibleMultipleReceive() {
    worker a {
        int v = 1;
        v -> function;
    }
    worker b {
        string v = "b";
        v -> function;
    }
    record {|int a; int b;|} _ = <- {a, b}; // @error b is a string
}
