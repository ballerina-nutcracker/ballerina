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

type E error<map<anydata|readonly>>;

class Emit {
    int n = 0;

    public isolated function next() returns record {| int value; |}|() {
        if self.n >= 1 {
            return ();
        }
        self.n += 1;
        return {value: 1};
    }
}

class StrEmit {
    public isolated function next() returns record {| string value; |}|() {
        return ();
    }
}

function toParameterized(stream s) returns stream<any|error, error?> {
    stream<any|error, error?> t = s;
    return t;
}

function toBare(stream<any|error, error?> s) returns stream {
    stream t = s;
    return t;
}

function toAliased(stream s) returns stream<any|error, E?> {
    stream<any|error, E?> t = s;
    return t;
}

function isParameterized(any v) returns boolean {
    return v is stream<any|error, error?>;
}

function isValueAny(any v) returns boolean {
    return v is stream<any, error?>;
}

function closeNonInt(stream s) returns error? {
    if s is stream<int> {
        return;
    }
    error? c = s.close();
    return c;
}

public function main() {
    stream bare = new (new Emit());
    io:println(isParameterized(bare)); // @output true
    io:println(isValueAny(bare)); // @output false
    io:println(isParameterized(toBare(toParameterized(bare)))); // @output true
    io:println(isParameterized(toAliased(bare))); // @output true
    io:println(toAliased(bare).next()); // @output {"value":1}
    stream<string, ()> strs = new (new StrEmit());
    io:println(closeNonInt(strs)); // @output
}
