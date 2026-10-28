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

isolated int counter = 0;

int mutableGlobal = 0;

function notIsolated() returns int {
    return mutableGlobal;
}

function forkInLock() {
    lock {
        counter += 1;
        fork { // @error cannot use a fork statement inside a lock statement
            worker w {
            }
        }
    }
}

function forkInLambdaInLock() {
    lock {
        counter += 1;
        isolated function () f = isolated function () {
            fork { // @error cannot use a fork statement inside a lock statement
                worker w {
                }
            }
        };
        f();
    }
}

function forkInLockInLambdaInLock() {
    lock {
        counter += 1;
        isolated function () f = isolated function () {
            lock {
                counter += 1;
                fork { // @error cannot use a fork statement inside a lock statement
                    worker w {
                    }
                }
            }
        };
        f();
    }
}

isolated function run(isolated function () f) {
    f();
}

function forkInIsolatedArgumentInLock() {
    lock {
        counter += 1;
        run(isolated function () {
            fork { // @error cannot use a fork statement inside a lock statement
                worker w {
                }
            }
        });
    }
}

function forkWorkerCallsNonIsolated() {
    lock {
        counter += 1;
        fork { // @error cannot use a fork statement inside a lock statement
            worker w returns int {
                return notIsolated(); // @error invocation of a non-isolated function inside lock statement
            }
        }
    }
}
