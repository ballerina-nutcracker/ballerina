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

function initStatements() returns int {
    fork {
        worker f returns int {
            return 1;
        }
    }
    worker w returns int {
        return 2;
    }
    int a = wait f;
    int b = wait w;
    return a + b;
}

function trailingStatements() returns int {
    worker w returns int {
        return 3;
    }
    int a = wait w;
    fork {
        worker f returns int {
            return a + 1;
        }
    }
    return wait f;
}

function nestedFork() returns int {
    fork {
        worker host returns int {
            fork {
                worker inner returns int {
                    return 5;
                }
            }
            return wait inner;
        }
    }
    return wait host;
}

function inLambda() returns int {
    var g = function() returns int {
        fork {
            worker f returns int {
                return 6;
            }
        }
        return wait f;
    };
    return g();
}

class Counter {
    int base = 7;

    function next() returns int {
        int b = self.base;
        fork {
            worker f returns int {
                return b;
            }
        }
        return wait f;
    }
}

function afterIfWithoutElse(boolean flag) returns int {
    if flag {
        return 0;
    }
    fork {
        worker f returns int {
            return 9;
        }
    }
    return wait f;
}

isolated function inIsolated(int x) returns int {
    fork {
        worker a returns int {
            return x;
        }
        worker b returns int {
            return x * 2;
        }
    }
    map<int> r = wait {a, b};
    return r.get("a") + r.get("b");
}

public function main() {
    io:println(initStatements()); // @output 3
    io:println(trailingStatements()); // @output 4
    io:println(nestedFork()); // @output 5
    io:println(inLambda()); // @output 6
    Counter c = new;
    io:println(c.next()); // @output 7
    io:println(afterIfWithoutElse(false)); // @output 9
    io:println(inIsolated(10)); // @output 30
}
