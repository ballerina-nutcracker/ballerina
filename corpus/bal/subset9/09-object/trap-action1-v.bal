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

client class Client {
    remote function value() returns int {
        return 1;
    }

    resource function get albums() returns int {
        return 1;
    }

    remote function broken() returns int {
        panic error("remote panic");
    }

    resource function get broken() returns int {
        panic error("resource panic");
    }

    remote function returnError(error failure) returns int|error {
        return failure;
    }

    resource function get failure(error failure) returns int|error {
        return failure;
    }
}

public function main() {
    Client c = new;
    int|error remoteValue = trap c->value();
    io:println(remoteValue); // @output 1
    int|error resourceValue = trap c->/albums;
    io:println(resourceValue); // @output 1

    int|error remotePanic = trap c->broken();
    if remotePanic is error {
        io:println(remotePanic.message()); // @output remote panic
    }
    int|error resourcePanic = trap c->/broken;
    if resourcePanic is error {
        io:println(resourcePanic.message()); // @output resource panic
    }

    error failure = error("returned error");
    var remoteError = trap c->returnError(failure);
    io:println(remoteError === failure); // @output true
    var resourceError = trap c->/failure.get(failure);
    io:println(resourceError === failure); // @output true

    var nested = trap trap c->broken();
    if nested is error {
        io:println(nested.message()); // @output remote panic
    }
    io:println("continued"); // @output continued
}
