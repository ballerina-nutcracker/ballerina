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
import ballerina/time;

listener http:Listener defaults = new (19254);
listener http:Listener never2 = new (19255, {http1Settings: {keepAlive: http:KEEPALIVE_NEVER}});
listener http:Listener never1 = new (19256, {httpVersion: http:HTTP_1_1, http1Settings: {keepAlive: http:KEEPALIVE_NEVER}});
listener http:Listener secure = new (19257, {
    secureSocket: {key: {certFile: "testdata/certs/server.crt", keyFile: "testdata/certs/server.key"}},
    http2MaxActiveStreams: 9,
    http2InitialWindowSize: 80000
});
listener http:Listener stopping = new (19258, {gracefulStopTimeout: 0.3});

function h1Connection(int port, string version) returns string|error = external;

function h2Settings(int port, boolean secure) returns string|error = external;

service / on defaults, never2, never1, secure {
    resource function get hello() returns string => "hello";
}

isolated boolean slowStarted = false;

service / on stopping {
    resource function get slow() returns string {
        lock {
            slowStarted = true;
        }
        runtime:sleep(3);
        return "slow";
    }
}

// Isolated, since a strand started from a non-isolated function does not run until it is waited for (#1758).
isolated function fetchSlow() returns string|error {
    http:Client c = check new ("http://localhost:19258");
    return c->/slow;
}

public function testMain() returns error? {
    io:println(check h1Connection(19254, "HTTP/1.1")); // @output connection= connection=
    io:println(check h1Connection(19255, "HTTP/1.1")); // @output connection=close closed
    io:println(check h1Connection(19256, "HTTP/1.1")); // @output connection=close closed
    io:println(check h1Connection(19254, "HTTP/1.0")); // @output connection=keep-alive connection=keep-alive
    io:println(check h1Connection(19255, "HTTP/1.0")); // @output connection=close closed
    io:println(check h2Settings(19254, false)); // @output maxConcurrentStreams=100 initialWindowSize=65535
    // An HTTP/2 connection to a listener whose HTTP/1.x connections never stay alive is unaffected.
    io:println(check h2Settings(19255, false)); // @output maxConcurrentStreams=100 initialWindowSize=65535
    io:println(check h2Settings(19257, true)); // @output maxConcurrentStreams=9 initialWindowSize=80000

    future<string|error> slow = start fetchSlow();
    boolean started = false;
    int polls = 0;
    while !started && polls < 200 {
        runtime:sleep(0.05);
        lock {
            started = slowStarted;
        }
        polls += 1;
    }
    time:Utc before = time:utcNow();
    error? stopped = stopping.gracefulStop();
    decimal waited = time:utcDiffSeconds(time:utcNow(), before);
    // @output Listener.gracefulStop: context deadline exceeded true
    io:println(stopped is error ? stopped.message() : "stopped", " ", waited < 2d);
    string|error slowResult = wait slow;
    io:println(slowResult is string ? slowResult : slowResult.message()); // @output slow
}
