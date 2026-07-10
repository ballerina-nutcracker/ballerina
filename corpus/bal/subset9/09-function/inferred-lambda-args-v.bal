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

var moduleLambda = function(int value = 5) returns int {
    return value;
};
var moduleLambdaAlias = moduleLambda;

public function main() {
    io:println(moduleLambda()); // @output 5
    io:println(moduleLambda(value = 6)); // @output 6
    io:println(moduleLambdaAlias()); // @output 5
    io:println(moduleLambdaAlias(value = 7)); // @output 7

    var lambda = function(int value = 5) returns int {
        return value;
    };
    io:println(lambda()); // @output 5
    io:println(lambda(value = 8)); // @output 8

    var lambdaAlias = lambda;
    io:println(lambdaAlias()); // @output 5
    io:println(lambdaAlias(value = 9)); // @output 9

    function(int x = 10) returns int typedLambda = lambda;
    io:println(typedLambda()); // @output 10
    io:println(typedLambda(x = 11)); // @output 11

    var includedRecordLambda = function(*Pair pair) returns int {
        return pair.left + pair.right;
    };
    io:println(includedRecordLambda(left = 1, right = 2)); // @output 3

    var includedRecordLambdaAlias = includedRecordLambda;
    io:println(includedRecordLambdaAlias(left = 3, right = 4)); // @output 7

    var groupedLambda = (function(int value = 5) returns int {
        return value;
    });
    io:println(groupedLambda()); // @output 5
    io:println(groupedLambda(value = 10)); // @output 10
}
