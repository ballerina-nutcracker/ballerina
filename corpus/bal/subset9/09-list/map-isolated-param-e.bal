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
        return values.map(nonIsolated).length(); // @error
    }
}

isolated function invalidNestedInferredLambda(int[] values) returns int[] {
    return values.map(value => [value].map(nonIsolated)[0]); // @error
}
