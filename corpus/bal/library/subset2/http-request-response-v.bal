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
    // ---- Request payloads ----
    http:Request req = new;
    req.method = "POST";
    req.rawPath = "/search?q=ballerina&tag=lang&tag=go";

    req.setTextPayload("body text");
    io:println(check req.getTextPayload()); // @output body text
    io:println(req.getContentType());       // @output text/plain

    // ---- Request headers ----
    req.setHeader("X-One", "v1");
    req.addHeader("X-One", "v2");
    io:println(check req.getHeader("X-One"));            // @output v1
    string[] xs = check req.getHeaders("X-One");
    io:println(xs.length());                             // @output 2
    io:println(req.hasHeader("X-One"));                  // @output true
    io:println(req.hasHeader("X-Absent"));               // @output false

    check req.setContentType("application/json");
    io:println(req.getContentType());                    // @output application/json

    req.removeHeader("X-One");
    io:println(req.hasHeader("X-One"));                  // @output false

    // ---- Request query params ----
    // A client-constructed Request has no parsed query string ($queryStr is set
    // server-side), so these return empty results but still exercise the natives.
    map<string[]> qp = req.getQueryParams();
    io:println(qp.length());                             // @output 0
    io:println(req.getQueryParamValue("q") is ());       // @output true
    io:println(req.getQueryParamValues("tag") is ());    // @output true

    // ---- Request JSON / binary payloads ----
    req.setJsonPayload({"k": "v"});
    json jp = check req.getJsonPayload();
    io:println(jp);                                      // @output {"k":"v"}

    req.setBinaryPayload([1, 2, 3]);
    byte[] bp = check req.getBinaryPayload();
    io:println(bp.length());                             // @output 3

    // ---- Response ----
    http:Response res = new;
    res.statusCode = 201;
    res.setHeader("X-Resp", "r1");
    res.addHeader("X-Resp", "r2");
    io:println(res.statusCode);                          // @output 201
    io:println(res.hasHeader("X-Resp"));                 // @output true
    io:println(check res.getHeader("X-Resp"));           // @output r1
    string[] rs = check res.getHeaders("X-Resp");
    io:println(rs.length());                             // @output 2
    io:println(res.getHeaderNames().length() >= 1);      // @output true

    check res.setContentType("text/plain");
    io:println(res.getContentType());                    // @output text/plain
    res.setTextPayload("resp body");
    io:println(check res.getTextPayload());              // @output resp body

    res.removeHeader("X-Resp");
    io:println(res.hasHeader("X-Resp"));                 // @output false
    res.removeAllHeaders();
    io:println(res.hasHeader("Content-Type"));           // @output false
}
