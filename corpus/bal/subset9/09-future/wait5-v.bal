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

function number() returns int {
    return 42;
}

function text() returns string {
    return "done";
}

function failure() returns error {
    return error("failed");
}

function identity(future<string> value) returns future<string> {
    return value;
}

public function main() {
    future<int> numberFuture = start number();
    future<string> textFuture = start text();
    future<error> failureFuture = start failure();
    record {|
        int|error numberFuture;
        string|error label;
        error failureFuture;
    |} result = wait {numberFuture, label: identity(textFuture), failureFuture};
    io:println(result.numberFuture); // @output 42
    io:println(result.label); // @output done
    io:println(result.failureFuture.message()); // @output failed

    future<int> repeated = start number();
    var repeatedResult = wait {first: repeated, second: repeated};
    io:println(repeatedResult["first"]); // @output 42
    io:println(repeatedResult["second"]); // @output error("multiple waits on the same future is not allowed")

    future<int> inferredNumber = start number();
    future<string> inferredText = start text();
    var inferred = wait {number: inferredNumber, text: inferredText};
    int|error inferredNumberResult = inferred.number;
    string|error inferredTextResult = inferred.text;
    io:println(inferredNumberResult); // @output 42
    io:println(inferredTextResult); // @output done

    future<int> mapped = start number();
    map<int|error> mappedResult = wait {value: mapped};
    mappedResult["extra"] = 2;
    io:println(mappedResult["extra"]); // @output 2

    future<int> open = start number();
    record {|
        int|error value;
        int|error...;
    |} openResult = wait {value: open};
    openResult["extra"] = 3;
    io:println(openResult["extra"]); // @output 3
}
