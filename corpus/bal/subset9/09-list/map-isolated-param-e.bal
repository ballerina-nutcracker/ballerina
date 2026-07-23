// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/lang.array;

function nonIsolated(int value) returns int {
    return value + 1;
}

isolated function invalidMethodCall(int[] values) returns int[] {
    return values.map(nonIsolated); // @error
}

isolated function invalidNamedCall(int[] values) returns int[] {
    return array:map(arr = values, func = nonIsolated); // @error
}

function invalidLockCall(int[] values) returns int {
    lock {
        int explicitLength = values.map(nonIsolated).length(); // @error
        int inferredLength = values.map((value) => nonIsolated(value)).length(); // @error
        return explicitLength + inferredLength;
    }
}

type InvalidRecord record {|
    int[] values = [1].map(nonIsolated); // @error
|};

isolated function invalidDefault(int[] values = [1].map(nonIsolated)) returns int { // @error
    return values[0];
}

function createInvalidIsolatedLambda() returns (isolated function() returns int) {
    return isolated function() returns int {
        return [1].map(func = nonIsolated).length(); // @error
    };
}

function invalidateNarrowingCapturedByCallback() returns int {
    int|string value = 1;
    if value is int {
        int[] _ = [1].map(function(int item) returns int {
            value = "changed";
            return item;
        });
        return value; // @error
    }
    return 0;
}
