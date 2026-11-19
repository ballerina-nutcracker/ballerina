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

import ballerina/http;
import ballerina/io;
import ballerina/lang.runtime;

// Sends a request whose chunked body is cut off, then closes the connection.
function abortUpload(int port, string path, string contentType) = external;

isolated string[] results = [];

isolated function note(any|error result) {
    string outcome = result is http:GenericClientError ? "GenericClientError: " + result.message()
        : result is error ? "error: " + result.message() : "ok";
    lock {
        results.push(outcome);
    }
}

isolated function awaitResult(int count) returns string {
    int waited = 0;
    while waited < 100 {
        lock {
            if results.length() >= count {
                return results[count - 1];
            }
        }
        runtime:sleep(0.05);
        waited += 1;
    }
    return "timed out";
}

service /read on new http:Listener(19228) {
    resource function post text(http:Request req) {
        note(req.getTextPayload());
    }

    resource function post 'json(http:Request req) {
        note(req.getJsonPayload());
    }

    resource function post 'xml(http:Request req) {
        note(req.getXmlPayload());
    }

    resource function post binary(http:Request req) {
        note(req.getBinaryPayload());
    }

    resource function post parts(http:Request req) {
        note(req.getBodyParts());
    }
}

public function testMain() {
    abortUpload(19228, "/read/text", "text/plain");
    io:println(awaitResult(1)); // @output GenericClientError: Error occurred while retrieving the text payload from the request
    abortUpload(19228, "/read/json", "application/json");
    io:println(awaitResult(2)); // @output GenericClientError: Error occurred while retrieving the json payload from the request
    abortUpload(19228, "/read/xml", "application/xml");
    io:println(awaitResult(3)); // @output GenericClientError: Error occurred while retrieving the xml payload from the request
    abortUpload(19228, "/read/binary", "application/octet-stream");
    io:println(awaitResult(4)); // @output GenericClientError: Error occurred while retrieving the binary payload from the request
    abortUpload(19228, "/read/parts", "multipart/form-data; boundary=b");
    io:println(awaitResult(5)); // @output GenericClientError: Error occurred while retrieving body parts from the request
}
