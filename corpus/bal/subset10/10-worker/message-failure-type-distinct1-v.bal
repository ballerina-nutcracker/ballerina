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

type AErr distinct error;

type BErr distinct error;

function failA(boolean flag) returns AErr? {
    return flag ? error AErr("a failed") : ();
}

function failB(boolean flag) returns BErr? {
    return flag ? error BErr("b failed") : ();
}

function receives(boolean failing) returns string {
    worker a returns AErr? {
        check failA(failing);
        1 -> function;
    }
    worker b returns BErr? {
        check failB(failing);
        2 -> function;
    }
    int|AErr x = <- a;
    record {|int b;|}|BErr r = <- {b};
    return (x is AErr ? x.message() : x.toString()) + " " + (r is BErr ? r.message() : r.toString());
}

function sends() returns boolean {
    worker a returns AErr? {
        check failA(false);
        int _ = <- function;
        int _ = <- function;
    }
    1 -> a;
    AErr? flushed = flush a;
    AErr? sent = 2 ->> a;
    return flushed is () && sent is ();
}

function fromDefault(boolean failing) returns string|BErr {
    worker a returns string {
        int|BErr r = <- function;
        return r is BErr ? r.message() : r.toString();
    }
    check failB(failing);
    5 -> a;
    return wait a;
}

public function main() {
    io:println(receives(false)); // @output 1 {"b":2}
    io:println(receives(true)); // @output a failed b failed
    io:println(sends()); // @output true
    io:println(fromDefault(false)); // @output 5
}
