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

class SlowParts {
    private int sent = 0;

    public isolated function next() returns record {|byte[] value;|}|io:Error? {
        if self.sent == 8 {
            return;
        }
        if self.sent > 0 {
            runtime:sleep(0.1);
        }
        self.sent += 1;
        return {value: "part".toBytes()};
    }
}

class Big {
    private int sent = 0;

    public isolated function next() returns record {|byte[] value;|}|io:Error? {
        if self.sent == 1000 {
            return;
        }
        self.sent += 1;
        byte[] block = [];
        block[65535] = 120;
        return {value: block};
    }
}

public function main() returns error? {
    // The test server speaks HTTP/1.1 only.
    http:Client c = check new ("http://testserver", {httpVersion: http:HTTP_1_1, timeout: 1});
    http:Response|error result = c->get("/delay");
    io:println(result is error ? result.message() : result.statusCode); // @output Idle timeout triggered before initiating inbound response

    // An in-memory upload the server reads slowly keeps the request alive while it keeps flowing.
    byte[] payload = [];
    payload[16777215] = 120;
    string uploaded = check c->post("/slow-read", payload);
    io:println(uploaded); // @output 16777216

    // Upload reads after the response has arrived do not restart the timeout for the response,
    // so its body can be read later than the timeout.
    http:Client quick = check new ("http://testserver", {httpVersion: http:HTTP_1_1, timeout: 0.3});
    http:Response early = check quick->post("/early", new stream<byte[], io:Error?>(new SlowParts()));
    runtime:sleep(0.6);
    io:println(early.getTextPayload()); // @output done

    // Each redirect hop gets the whole timeout, so a chain that takes longer than it succeeds.
    http:Client following = check new ("http://testserver", {httpVersion: http:HTTP_1_1, timeout: 0.25, followRedirects: {enabled: true}});
    string followed = check following->get("/hop/4");
    io:println(followed); // @output done

    // A body that stalls once it reaches maxEntityBodySize fails with the idle timeout.
    http:Client limited = check new ("http://testserver", {httpVersion: http:HTTP_1_1, timeout: 0.1, responseLimits: {maxEntityBodySize: 4}});
    http:Response stalled = check limited->get("/stall-body");
    string|error stalledBody = stalled.getTextPayload();
    io:println(stalledBody is error ? stalledBody.message() : stalledBody);
    // @output Idle timeout triggered while reading inbound response entity body

    // An upload the server stops reading after a complete response is ended by the timeout.
    http:Client uploader = check new ("http://testserver", {httpVersion: http:HTTP_1_1, timeout: 1});
    http:Response|error abandoned = uploader->post("/stall-upload", new stream<byte[], io:Error?>(new Big()));
    io:println(abandoned is error ? abandoned.message() : abandoned.getTextPayload()); // @output ok
}
