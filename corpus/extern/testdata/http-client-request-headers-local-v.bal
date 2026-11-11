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
}
