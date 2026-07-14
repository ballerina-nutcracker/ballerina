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

type Pair record {|
    int left;
    int right;
|};

type F function(int a = 5) returns int;
type FAlias F;
type ChainedFAlias FAlias;

public function main() {
    (function(int x) returns int)|(function(float y) returns float) narrowed = function(int value) returns int {
        return value;
    };
    if narrowed is function(float y = 30) returns float {
        io:println(narrowed(y = 12));
    } else if narrowed is function(int z = 20) returns int {
        io:println(narrowed()); // @output 20
        io:println(narrowed(z = 11)); // @output 11
    }

    (function(Pair p) returns int)|int narrowedRecord = function(Pair value) returns int {
        return value.left + value.right;
    };
    if narrowedRecord is function(*Pair pair) returns int {
        io:println(narrowedRecord(left = 6, right = 7)); // @output 13
    }

    ChainedFAlias|float narrowedAlias = function(int value) returns int {
        return value;
    };
    if narrowedAlias is ChainedFAlias {
        io:println(narrowedAlias()); // @output 5
        io:println(narrowedAlias(a = 14)); // @output 14
    }

    (function(int x) returns int)|string negated = function(int value) returns int {
        return value;
    };
    if negated !is function(int n = 7) returns int {
        io:println("unexpected");
    } else {
        io:println(negated()); // @output 7
        io:println(negated(n = 16)); // @output 16
    }
}
