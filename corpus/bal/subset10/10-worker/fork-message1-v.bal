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

function asyncAndSync() {
    fork {
        worker a returns int {
            1 -> b;
            error? e = 2 ->> b;
            return e is () ? 1 : 0;
        }
        worker b returns int {
            int x = <- a;
            int y = <- a;
            return x + y;
        }
    }
    map<int> r = wait {a, b};
    io:println(r); // @output {"a":1,"b":3}
}

function multipleReceive() {
    fork {
        worker a {
            1 -> c;
        }
        worker b {
            "x" -> c;
        }
        worker c returns record {int a; string b;} {
            return <- {a, b};
        }
    }
    record {int a; string b;} r = wait c;
    io:println(r); // @output {"a":1,"b":"x"}
}

function flushes() {
    fork {
        worker a returns boolean {
            1 -> b;
            2 -> c;
            error? one = flush b;
            error? all = flush;
            return one is () && all is ();
        }
        worker b returns int {
            return <- a;
        }
        worker c returns int {
            return <- a;
        }
    }
    boolean flushed = wait a;
    map<int> r = wait {b, c};
    io:println(flushed); // @output true
    io:println(r); // @output {"b":1,"c":2}
}

function senderFails() {
    fork {
        worker a returns error? {
            int x = check int:fromString("not a number");
            return x ->> b;
        }
        worker b returns int|error {
            int|error x = <- a;
            return x;
        }
    }
    int|error r = wait b;
    io:println(r is error ? r.message() : "received"); // @output 'int' from string: invalid number format: not a number
    error? sent = wait a;
    io:println(sent is error); // @output true
}

function receiverFails() {
    fork {
        worker a returns boolean {
            error? r = 1 ->> b;
            if r is error {
                io:println(r.message()); // @output 'int' from string: invalid number format: bad
            }
            return r is error;
        }
        worker b returns error? {
            _ = check int:fromString("bad");
            int _ = <- a;
        }
    }
    boolean failed = wait a;
    error? received = wait b;
    io:println(failed && received is error); // @output true
}

public function main() {
    asyncAndSync();
    multipleReceive();
    flushes();
    senderFails();
    receiverFails();
}
