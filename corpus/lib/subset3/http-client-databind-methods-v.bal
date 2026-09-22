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

public function main() returns error? {
    http:Client c = check new ("https://example.com");

    // Every remote method except `head` binds the response payload.
    string postBody = check c->post("/path", "req");
    io:println(postBody); // @output test body

    string putBody = check c->put("/path", "req");
    io:println(putBody); // @output test body

    string patchBody = check c->patch("/path", "req");
    io:println(patchBody); // @output test body

    string deleteBody = check c->delete("/path");
    io:println(deleteBody); // @output test body

    string optionsBody = check c->options("/path");
    io:println(optionsBody); // @output test body

    string execBody = check c->execute("PATCH", "/path", "req");
    io:println(execBody); // @output test body

    http:Request req = new;
    req.setTextPayload("fwd");
    string forwardBody = check c->forward("/path", req);
    io:println(forwardBody); // @output test body

    // `head` has no targetType parameter and always returns the response.
    http:Response headResp = check c->head("/path");
    io:println(headResp.statusCode); // @output 200

    // Binding still applies when headers and mediaType are supplied.
    byte[] withHeaders = check c->post("/path", "req", {"X-Custom": "value"}, "text/plain");
    io:println(withHeaders.length()); // @output 9

    return;
}
