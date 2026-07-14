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
    http:Client c = check new http:Client("http://testserver", {});

    // Forward a request whose body is materialised first and that carries a
    // hop-by-hop header (stripped) alongside a normal header (forwarded).
    // Exercises materialize, takeStream, and removeHopByHopHeaders.
    http:Request req = new;
    req.method = "POST";
    req.setTextPayload("forwarded body");
    io:println(check req.getTextPayload()); // @output forwarded body
    req.setHeader("Connection", "keep-alive");
    req.setHeader("X-Keep", "kept");
    http:Response fr = check c->forward("/fwd", req);
    io:println(fr.statusCode); // @output 200

    // GET with mixed single- and multi-valued request headers -> extractHeaders.
    map<string|string[]> headers = {"X-Single": "one", "X-Multi": ["a", "b"]};
    http:Response hr = check c->get("/h", headers);
    io:println(hr.statusCode); // @output 200

    // Compression ALWAYS adds Accept-Encoding; NEVER strips it
    // -> applyCompressionHeaders / compressionModeOf.
    http:Client always = check new http:Client("http://testserver", {compression: http:COMPRESSION_ALWAYS});
    http:Response ar = check always->get("/h");
    io:println(ar.statusCode); // @output 200
    http:Client noComp = check new http:Client("http://testserver", {compression: http:COMPRESSION_NEVER});
    http:Response nr = check noComp->get("/h");
    io:println(nr.statusCode); // @output 200

    // A 204 No Content response has no body stream, exercising the empty-body
    // response holder (newResponseBodyHolder with a nil stream).
    http:Response empty = check c->head("/empty");
    io:println(empty.statusCode); // @output 204

    // When the caller already set Accept-Encoding: ALWAYS leaves it untouched,
    // NEVER strips it (exercises both applyCompressionHeaders branches).
    map<string|string[]> aeHeader = {"Accept-Encoding": "identity"};
    http:Response ar2 = check always->get("/h", aeHeader);
    io:println(ar2.statusCode); // @output 200
    http:Response nr2 = check noComp->get("/h", aeHeader);
    io:println(nr2.statusCode); // @output 200
}
