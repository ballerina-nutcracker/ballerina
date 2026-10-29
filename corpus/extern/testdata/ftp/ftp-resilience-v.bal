// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

import ballerina/ftp;
import ballerina/io;
import ballerina/lang.runtime;

function ftpPort() returns int = external;

function busyPort() returns int = external;

function busyReplies() returns int = external;

function scrub(string message) returns string {
    string port = ":" + ftpPort().toString();
    string result = message;
    int? i = result.indexOf(port);
    while i is int {
        result = result.substring(0, i) + ":PORT" + result.substring(i + port.length());
        i = result.indexOf(port);
    }
    return result;
}

int ticks = 0;

function ticker() {
    foreach int _ in 0 ..< 20 {
        ticks += 1;
        runtime:sleep(0.02);
    }
}

function connect(ftp:CircuitBreakerConfig? circuitBreaker = (), ftp:RetryConfig? retryConfig = ())
        returns ftp:Client|ftp:Error {
    ftp:ClientConfiguration config = {
        host: "127.0.0.1",
        port: ftpPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}}
    };
    if circuitBreaker is ftp:CircuitBreakerConfig {
        config.circuitBreaker = circuitBreaker;
    }
    if retryConfig is ftp:RetryConfig {
        config.retryConfig = retryConfig;
    }
    return new (config);
}

function describe(any|error result) returns string {
    if result is ftp:CircuitBreakerOpenError {
        return "CircuitBreakerOpenError: " + result.message();
    }
    if result is ftp:AllRetryAttemptsFailedError {
        return "AllRetryAttemptsFailedError: " + scrub(result.message());
    }
    if result is error {
        return "Error: " + scrub(result.message());
    }
    return "ok";
}

public function main() returns error? {
    ftp:RetryConfig fastRetry = {count: 2, interval: 0.01, backOffFactor: 3.0, maxWaitInterval: 0.02};
    ftp:Client retrying = check connect(retryConfig = fastRetry);
    var missingText = retrying->getText("/missing.txt");
    io:println(describe(missingText));
    // @output Error: Failed to read file: ftp://ballerina:***@127.0.0.1:PORT/missing.txt not found
    io:println(missingText is ftp:FileNotFoundError); // @output true
    var missingBytes = retrying->getBytes("/missing.txt");
    io:println(missingBytes is ftp:FileNotFoundError); // @output true
    check retrying->putText("/bad.json", "{");
    json|ftp:Error badJson = retrying->getJson("/bad.json");
    io:println(badJson is ftp:Error && !(badJson is ftp:AllRetryAttemptsFailedError)); // @output true
    xml|ftp:Error missingXml = retrying->getXml("/missing.xml");
    io:println(missingXml is ftp:FileNotFoundError); // @output true
    check retrying->putText("/ok.txt", "fine");
    var okText = retrying->getText("/ok.txt");
    io:println(okText); // @output fine
    check retrying->close();

    ftp:Client busy = check new ({host: "127.0.0.1", port: busyPort(), retryConfig: fastRetry});
    var busyText = busy->getText("/busy.txt");
    io:println(busyText is ftp:AllRetryAttemptsFailedError); // @output true
    string busyMessage = busyText is error ? busyText.message() : "";
    io:println(busyMessage.indexOf("Operation 'getText' failed after 2 retry attempts: ") == 0); // @output true
    io:println(busyReplies()); // @output 3
    check busy->close();

    ftp:Client busyOnce = check new ({
        host: "127.0.0.1",
        port: busyPort(),
        auth: {credentials: {username: "linger", password: "x"}}
    });
    var busyOnceText = busyOnce->getText("/busy.txt");
    io:println(busyOnceText is ftp:ServiceUnavailableError); // @output true
    check busyOnce->close();

    ftp:Client slowBusy = check new ({
        host: "127.0.0.1",
        port: busyPort(),
        retryConfig: {count: 2, interval: 0.3, backOffFactor: 1.0}
    });
    future<()> tick = start ticker();
    var slowText = slowBusy->getText("/busy.txt");
    io:println(slowText is ftp:AllRetryAttemptsFailedError, " ", ticks >= 10); // @output true true
    check wait tick;
    check slowBusy->close();

    ftp:CircuitBreakerConfig breaker = {
        rollingWindow: {requestVolumeThreshold: 2, timeWindow: 10, bucketSize: 1},
        failureThreshold: 0.4,
        resetTime: 1,
        failureCategories: [ftp:ALL_ERRORS]
    };
    ftp:Client guarded = check connect(breaker, fastRetry);
    check guarded->putText("/present.txt", "here");
    var first = guarded->getText("/missing.txt");
    io:println(first is ftp:FileNotFoundError); // @output true
    var opened = guarded->exists("/present.txt");
    io:println(describe(opened)); // @output CircuitBreakerOpenError: FTP server unavailable. Circuit breaker is OPEN. Retry in 1 seconds.
    io:println(opened is ftp:ServiceUnavailableError); // @output true
    var stillOpen = guarded->getText("/present.txt");
    io:println(stillOpen is ftp:CircuitBreakerOpenError); // @output true
    runtime:sleep(1.1);
    var trial = guarded->getText("/present.txt");
    io:println(trial); // @output here
    var closedAgain = guarded->exists("/present.txt");
    io:println(closedAgain); // @output true
    check guarded->close();

    ftp:CircuitBreakerConfig connectionOnly = {
        rollingWindow: {requestVolumeThreshold: 1, timeWindow: 10, bucketSize: 5},
        failureThreshold: 0.0,
        failureCategories: [ftp:CONNECTION_ERROR, ftp:AUTHENTICATION_ERROR, ftp:TRANSIENT_ERROR]
    };
    ftp:Client selective = check connect(connectionOnly);
    var notFound = selective->getText("/missing.txt");
    io:println(notFound is ftp:FileNotFoundError); // @output true
    var stillClosed = selective->exists("/missing.txt");
    io:println(stillClosed); // @output false
    check selective->close();
}
