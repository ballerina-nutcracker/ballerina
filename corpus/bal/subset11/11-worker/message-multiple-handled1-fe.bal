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

function failWith(string message) returns error? {
    return error(message);
}

// A failed multiple receive takes none of its messages, so a worker that
// handles its error and then waits on another sender would deadlock with it.
function handledThenWait() returns int {
    worker a returns error? {
        check failWith("a failed");
        1 -> function;
    }
    worker b returns int {
        2 ->> function;
        return 5;
    }
    record {| int a; int b; |}|error r = <- {a, b}; // @error the error of a multiple receive must end the worker
    boolean _ = r is error;
    int x = wait b;
    return x;
}

function trapped() {
    worker a returns error? {
        check failWith("a failed");
        1 -> function;
    }
    worker b returns error? {
        check failWith("b failed");
        2 -> function;
    }
    record {int a; int b;}|error r = trap <- {a, b}; // @error trap handles the error
    boolean _ = r is error;
}

function trappedCheckPanic() {
    worker a returns error? {
        check failWith("a failed");
        1 -> function;
    }
    worker b returns error? {
        2 ->> function;
    }
    record {int a; int b;}|error r = trap checkpanic <- {a, b}; // @error trap also catches checkpanic's panic
    boolean _ = r is error;
}

function panics() {
    panic error("a panicked");
}

// Neither sender can fail, but trap would catch a's panic and leave b's
// message unreceived.
function trappedPanic() {
    worker a {
        panics();
        1 -> function;
    }
    worker b {
        2 ->> function;
    }
    record {int a; int b;}|error r = trap <- {a, b}; // @error trap catches a sender's panic
    boolean _ = r is error;
}

function matched() {
    worker a returns error? {
        check failWith("a failed");
        1 -> function;
    }
    worker b returns error? {
        check failWith("b failed");
        2 -> function;
    }
    match <- {a, b} { // @error a match handles the error
        _ => {
        }
    }
}

public function main() {
    _ = handledThenWait();
    trapped();
    trappedCheckPanic();
    trappedPanic();
    matched();
}
