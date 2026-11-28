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

function waitForms() {
    fork {
        worker a returns int {
            return 1;
        }
        worker b returns int {
            return 2;
        }
        worker c returns int {
            return 3;
        }
        worker d returns string {
            return "d";
        }
        worker e returns string {
            runtime:sleep(0.5);
            return "e";
        }
    }
    int one = wait a;
    io:println(one); // @output 1
    map<int> both = wait {b, c};
    io:println(both); // @output {"b":2,"c":3}
    string s = wait d | e;
    io:println(s); // @output d
}

// The fork doesn't join its workers: the statements after it run while the
// workers are still running.
function noJoin() {
    fork {
        worker slow returns string {
            runtime:sleep(0.2);
            io:println("worker done");
            return "slow";
        }
    }
    io:println("after fork"); // @output after fork
    string r = wait slow; // @output worker done
    io:println(r); // @output slow
}

function noReturnsClause() {
    fork {
        worker n {
            io:println("in n"); // @output in n
        }
    }
    () r = wait n;
    io:println(r is ()); // @output true
}

function capturesBeforeFork(int param) {
    worker named returns int {
        return param;
    }
    int trailing = param + 10;
    fork {
        worker f returns int {
            return param + trailing;
        }
    }
    int n = wait named;
    int r = wait f;
    io:println(n + r); // @output 19
}

public function main() {
    waitForms();
    noJoin();
    noReturnsClause();
    capturesBeforeFork(3);
}
