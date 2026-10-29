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

isolated function isolatedFn(int n) returns int {
    return n;
}

isolated function apply(isolated function () returns int f) returns int {
    return f();
}

function startInLambdaInLock() {
    isolated function () returns int f = isolated function () returns int {
        return 0;
    };
    lock {
        counter += 1;
        f = isolated function () returns int {
            future<int> r = start isolatedFn(42); // @error start action lexically inside lock
            return checkpanic wait r;
        };
    }
    _ = f;
}

function startInNestedLambdaInLock() {
    lock {
        counter += 1;
        _ = isolated function () returns int {
            isolated function () returns int g = isolated function () returns int {
                future<int> r = start isolatedFn(1); // @error start action lexically inside lock
                return checkpanic wait r;
            };
            return g();
        };
    }
}

function startInLambdaArgInLock() {
    lock {
        counter += apply(isolated function () returns int {
            future<int> r = start isolatedFn(2); // @error start action lexically inside lock
            return checkpanic wait r;
        });
    }
}

function startInLockOfLambdaInLock() {
    lock {
        counter += 1;
        _ = isolated function () {
            lock {
                counter += 1;
                future<int> r = start isolatedFn(3); // @error start action inside lock
                _ = r;
            }
        };
    }
}

isolated class Counter {
    private int n = 0;

    isolated function bump() {
        lock {
            self.n += 1;
            _ = isolated function () {
                future<int> r = start isolatedFn(4); // @error start action lexically inside lock
                _ = r;
            };
        }
    }
}

public function main() {
}
