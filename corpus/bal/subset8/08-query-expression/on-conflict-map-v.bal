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

int conflictHitCount = 0;

function onConflictNil() returns error? {
    conflictHitCount += 1;
    return ();
}

function onConflictError() returns error? {
    return error("duplicate key");
}

public function main() {
    int[] xs = [1, 2, 3];
    map<int>|error out1 = map from var x in xs
        select ["k", x]
        on conflict onConflictNil();
    map<int>|error out2 = map from var x in xs
        select ["k", x]
        on conflict onConflictError();
    map<decimal> contextualMap = map from var _ in [1]
        select ["k", 1];
    io:println(out1); // @output {"k":3}
    io:println(conflictHitCount); // @output 2
    io:println(out2 is error); // @output true
    io:println(contextualMap["k"] is decimal); // @output true
}
