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
import ballerina/test;

service /svc on new http:Listener(19240) {
    resource function get greeting() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload("Hello, World!");
        return resp;
    }

    resource function get boom() returns http:Response {
        test:assertFail("failed inside a resource");
    }
}

// Graceful-stop handlers only run during shutdown, after the last test has been
// reported, so this pins that a failure raised there still reaches the summary
// and the exit code.
public function main() {
    runtime:onGracefulStop(onStop);
}

function onStop() returns error? {
    error? trapped = trap test:assertFail("failure during graceful stop");
    if trapped is error {
        return;
    }
    return;
}

@test:Config {}
function testCallsTheService() returns error? {
    http:Client c = check new http:Client("http://localhost:19240", {});
    http:Response r = check c->get("/svc/greeting");
    if r.statusCode != 200 {
        test:assertFail("unexpected status");
    }
    io:println(check r.getTextPayload());
}

@test:Config {}
function testResourceAssertFail() returns error? {
    http:Client c = check new http:Client("http://localhost:19240", {});
    http:Response _ = check c->get("/svc/boom");
}

@test:Config {}
function testStrandAssertFail() {
    future<()> f = start failInStrand();
    error? outcome = trap wait f;
    io:println(outcome is error);
}

function failInStrand() {
    test:assertFail("failed inside a strand");
}
