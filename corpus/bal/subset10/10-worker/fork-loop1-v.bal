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
import ballerina/lang.runtime;

function foreachEach() {
    foreach int i in 0 ..< 3 {
        fork {
            worker a returns int {
                i -> b;
                return i;
            }
            worker b returns int {
                int v = <- a;
                return v * 10;
            }
        }
        map<int> r = wait {a, b};
        io:println(r); // @output {"a":0,"b":0}
        // @output {"a":1,"b":10}
        // @output {"a":2,"b":20}
    }
}

// Later iterations send first; each message still reaches its own
// iteration's receiver.
function foreachOverlapping() {
    future<int>[] results = [];
    foreach int i in 0 ..< 3 {
        fork {
            worker a {
                runtime:sleep(<decimal>(3 - i) * 0.05);
                i -> b;
            }
            worker b returns int {
                int v = <- a;
                return v * 10;
            }
        }
        results.push(b);
    }
    foreach future<int> f in results {
        int|error r = wait f;
        io:println(r); // @output 0
        // @output 10
        // @output 20
    }
}

function whileEach() {
    int i = 0;
    while i < 2 {
        final int n = i;
        fork {
            worker a {
                n -> b;
            }
            worker b returns int {
                int v = <- a;
                return v + 100;
            }
        }
        int r = wait b;
        io:println(r); // @output 100
        // @output 101
        i += 1;
    }
}

function whileOverlapping() {
    future<int>[] results = [];
    int i = 0;
    while i < 3 {
        final int n = i;
        fork {
            worker a {
                runtime:sleep(<decimal>(3 - n) * 0.05);
                n -> b;
            }
            worker b returns int {
                int v = <- a;
                return v + 1000;
            }
        }
        results.push(b);
        i += 1;
    }
    foreach future<int> f in results {
        int|error r = wait f;
        io:println(r); // @output 1000
        // @output 1001
        // @output 1002
    }
}

function lambdaCalledRepeatedly() {
    var f = function(int x) returns int {
        fork {
            worker a {
                x -> b;
            }
            worker b returns int {
                int v = <- a;
                return v * 2;
            }
        }
        return wait b;
    };
    io:println(f(1) + f(2)); // @output 6
}

function inNamedWorkerLoop() {
    worker host returns int {
        int sum = 0;
        foreach int i in 1 ... 3 {
            fork {
                worker a {
                    i -> b;
                }
                worker b returns int {
                    return <- a;
                }
            }
            sum += wait b;
        }
        return sum;
    }
    int r = wait host;
    io:println(r); // @output 6
}

public function main() {
    foreachEach();
    foreachOverlapping();
    whileEach();
    whileOverlapping();
    lambdaCalledRepeatedly();
    inNamedWorkerLoop();
}
