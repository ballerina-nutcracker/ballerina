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

// The server echoes "<content-type> <x-test values> <x-multi values>".
public function main() returns error? {
    http:Client c = check new http:Client("http://testserver", {});

    http:Request postReq = new;
    postReq.setTextPayload("hi");
    postReq.setHeader("X-Test", "post");
    string post = check c->post("/echo", postReq);
    io:println(post); // @output text/plain [post] []

    // Headers owned by the transport are dropped; a header that Connection names is still sent, as in jBallerina.
    http:Request hopReq = new;
    hopReq.setTextPayload("hi");
    hopReq.setHeader("Connection", "keep-alive, X-Test");
    hopReq.setHeader("Upgrade", "h2c");
    hopReq.setHeader("X-Test", "hop");
    string hop = check c->post("/echo", hopReq, {"Keep-Alive": "timeout=5"});
    io:println(hop); // @output text/plain [hop] []

    // An explicit Connection: close is honoured; the server rejects the request without it.
    string close = check c->post("/echo", "hi", {"Connection": "Close", "X-Test": "conn-close"});
    io:println(close); // @output text/plain [conn-close] []

    // get/head/options drop the same transport-owned headers.
    string get = check c->get("/echo", {"Upgrade": "h2c", "Keep-Alive": "timeout=5", "content-type": "a/b", "X-Test": "get"});
    io:println(get); // @output a/b [get] []

    // HTTP/2 permits only TE: trailers, so any other TE is dropped.
    http:Request teReq = new;
    teReq.setTextPayload("hi");
    teReq.setHeader("TE", "gzip");
    teReq.setHeader("X-Test", "te");
    string te = check c->post("/echo", teReq);
    io:println(te); // @output text/plain [te] []

    // A single TE: trailers is kept; the server rejects the request without it.
    http:Request trailersReq = new;
    trailersReq.setTextPayload("hi");
    trailersReq.setHeader("TE", "trailers");
    trailersReq.setHeader("X-Test", "te-kept");
    string trailers = check c->post("/echo", trailersReq);
    io:println(trailers); // @output text/plain [te-kept] []

    // TE values HTTP/2 rejects, a mixed-case or repeated trailers, are dropped.
    http:Request mixedReq = new;
    mixedReq.setTextPayload("hi");
    mixedReq.setHeader("TE", "Trailers");
    mixedReq.addHeader("TE", "trailers");
    mixedReq.setHeader("X-Test", "te-mixed");
    string mixed = check c->post("/echo", mixedReq);
    io:println(mixed); // @output text/plain [te-mixed] []

    // A string[] header argument is appended to the request's own values.
    http:Request putReq = new;
    putReq.setTextPayload("hi");
    putReq.addHeader("X-Multi", "a");
    putReq.addHeader("X-Multi", "b");
    string put = check c->put("/echo", putReq, {"x-multi": ["c", "d"]});
    io:println(put); // @output text/plain [] [a b c d]

    // A string header argument replaces the request's value.
    http:Request patchReq = new;
    patchReq.setTextPayload("hi");
    patchReq.setHeader("X-Test", "req");
    patchReq.setHeader("X-Multi", "kept");
    string patch = check c->patch("/echo", patchReq, {"X-Test": "arg"});
    io:println(patch); // @output text/plain [arg] [kept]

    // mediaType takes precedence over a Content-Type header argument.
    http:Request deleteReq = new;
    deleteReq.setTextPayload("hi");
    deleteReq.setHeader("X-Test", "delete");
    string delete = check c->delete("/echo", deleteReq, {"Content-Type": "a/b"}, "application/custom");
    io:println(delete); // @output application/custom [delete] []

    http:Request execReq = new;
    execReq.setJsonPayload({});
    execReq.setHeader("X-Test", "execute");
    string exec = check c->execute("POST", "/echo", execReq, {"X-Multi": "arg"});
    io:println(exec); // @output application/json [execute] [arg]

    http:Request ctReq = new;
    ctReq.setTextPayload("hi");
    ctReq.setHeader("Content-Type", "x/y");
    string ct = check c->post("/echo", ctReq);
    io:println(ct); // @output x/y [] []

    // A Content-Type header argument replaces the payload's default.
    string text = check c->post("/echo", "hi", {"Content-Type": "a/b", "X-Test": "text"});
    io:println(text); // @output a/b [text] []

    // The last value of a string[] Content-Type argument is the one sent.
    string textArr = check c->post("/echo", "hi", {"Content-Type": ["a/b", "c/d"]});
    io:println(textArr); // @output c/d [] []

    http:Request arrReq = new;
    arrReq.setTextPayload("hi");
    string reqArr = check c->post("/echo", arrReq, {"Content-Type": ["application/json"]});
    io:println(reqArr); // @output application/json [] []
}
