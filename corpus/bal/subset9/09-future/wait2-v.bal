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

import ballerina/io;

function firstError() returns error {
    return error("first");
}

function lastError() returns error {
    return error("last");
}

function number() returns int {
    return 42;
}

function text() returns string {
    return "done";
}

function optionalError(boolean shouldFail) returns error? {
    if shouldFail {
        return error("optional");
    }
    return ();
}

function crash() returns int {
    panic error("ignored");
}

function identity(future<int> value) returns future<int> {
    return value;
}

public function main() {
    future<error> failed = start firstError();
    future<int> succeeded = start number();
    int|error firstSuccess = wait failed | succeeded;
    io:println(firstSuccess); // @output 42

    future<error> earlierError = start firstError();
    future<error> laterError = start lastError();
    error allFailed = wait laterError | earlierError;
    io:println(allFailed.message()); // @output last

    future<int> numberFuture = start number();
    future<string> textFuture = start text();
    int|string|error heterogeneous = wait numberFuture | textFuture;
    io:println(heterogeneous); // @output 42

    future<error?> optionalFailure = start optionalError(true);
    future<error?> optionalSuccess = start optionalError(false);
    error? optionalSuccessResult = wait optionalFailure | optionalSuccess;
    io:println(optionalSuccessResult is ()); // @output true

    future<error?> optionalFailureOne = start optionalError(true);
    future<error?> optionalFailureTwo = start optionalError(true);
    error? optionalFailureResult = wait optionalFailureOne | optionalFailureTwo;
    io:println(optionalFailureResult is error); // @output true

    future<int> expressionFirst = start number();
    future<int> expressionSecond = start number();
    int|error expressionResult = wait identity(expressionFirst) | identity(expressionSecond);
    io:println(expressionResult); // @output 42

    future<error> duplicate = start firstError();
    error duplicateResult = wait duplicate | duplicate;
    io:println(duplicateResult.message()); // @output first

    future<int> claimed = start number();
    int|error claimedFirst = wait claimed;
    _ = claimedFirst is int;
    int|error claimedResult = wait claimed | claimed;
    io:println(claimedResult); // @output error("multiple waits on the same future is not allowed")

    future<int> completed = start number();
    future<int> ignoredPanic = start crash();
    int|error completedResult = wait completed | ignoredPanic;
    io:println(completedResult); // @output 42
}
