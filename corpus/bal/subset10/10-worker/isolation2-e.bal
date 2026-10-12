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

isolated function capturesNonFinal() returns int {
    int x = 1;
    worker w returns int {
        return x; // @error captured variable is not final
    }
    return wait w;
}

isolated function capturesMutableFinal() returns int {
    final int[] arr = [1];
    worker w returns int {
        return arr[0]; // @error captured variable is not isolated
    }
    return wait w;
}

function capturesFromIsolatedLambda() returns int {
    int x = 1;
    isolated function () returns int f = isolated function () returns int {
        int y = 2;
        worker w returns int {
            return x + y; // @error captured variables are not final
        }
        return wait w;
    };
    return f();
}

isolated function capturesMutableParam(int[] p) returns int {
    worker w returns int {
        return p[0]; // @error captured parameter is not isolated
    }
    return wait w;
}

isolated function assignsCaptured() returns int {
    int x = 1;
    worker w {
        x = 2; // @error captured variable is not final
    }
    wait w;
    return x;
}

isolated function lambdaInWorker() returns int {
    int x = 1;
    worker w returns int {
        isolated function () returns int f = isolated function () returns int {
            return x; // @error captured by the lambda
        };
        return f();
    }
    return wait w;
}

isolated function workerInIsolatedLambda() returns int {
    int x = 1;
    isolated function () returns int f = isolated function () returns int {
        worker w returns int {
            return x; // @error captured by the worker
        }
        return wait w;
    };
    return f();
}

isolated function queryVarInWorker() returns int {
    worker w returns int[] {
        return from int[] z in [[1]]
            let isolated function () returns int g = isolated function () returns int {
                isolated function () returns int h = isolated function () returns int {
                    return z[0]; // @error query variable is not isolated
                };
                return h();
            }
            select g();
    }
    int[] r = wait w;
    return r[0];
}

isolated function capturesNarrowed(int[]|string p) returns int {
    final int[]|string v = p;
    worker w returns int {
        if v is int[] { // @error captured variable is not isolated
            return v[0]; // @error narrowed captured variable is not isolated
        }
        return 0;
    }
    return wait w;
}

isolated class Counter {
    private int n = 0;

    isolated function readsOutsideLock() returns int {
        worker w returns int {
            return self.n; // @error mutable field access outside lock
        }
        return wait w;
    }
}

public function main() {
    io:println(capturesNonFinal());
    io:println(capturesMutableFinal());
    io:println(capturesFromIsolatedLambda());
    io:println(capturesMutableParam([1]));
    io:println(assignsCaptured());
    io:println(lambdaInWorker());
    io:println(workerInIsolatedLambda());
    io:println(queryVarInWorker());
    io:println(capturesNarrowed([1]));
    Counter c = new;
    io:println(c.readsOutsideLock());
}
