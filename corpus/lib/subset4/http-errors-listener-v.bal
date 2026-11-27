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

public function main() {
    http:Listener|http:ListenerError missingCert = new (9561, {secureSocket: {key: {certFile: "missing-cert.pem", keyFile: "missing-key.pem"}}});
    if missingCert is http:ListenerError {
        io:println(missingCert is http:GenericListenerError, " ", missingCert.message()); // @output true Key file or server certificates file not found
    }

    http:Listener|http:ListenerError negativeTimeout = new (9562, {timeout: -1});
    if negativeTimeout is http:ListenerError {
        // @output true Idle timeout cannot be negative. If you want to disable the timeout please use value 0
        io:println(negativeTimeout is http:GenericListenerError, " ", negativeTimeout.message());
    }

    http:Listener|http:ListenerError negativeGracefulStop = new (9563, {gracefulStopTimeout: -1});
    if negativeGracefulStop is http:ListenerError {
        // @output true gracefulStop timeout cannot be negative
        io:println(negativeGracefulStop is http:GenericListenerError, " ", negativeGracefulStop.message());
    }

    // Too small for a float, but still negative.
    http:Listener|http:ListenerError tinyNegativeTimeout = new (9566, {timeout: -1e-400d});
    if tinyNegativeTimeout is http:ListenerError {
        io:println(tinyNegativeTimeout.message()); // @output Idle timeout cannot be negative. If you want to disable the timeout please use value 0
    }

    // Unlike jBallerina, which accepts them, HTTP/2 settings net/http cannot advertise are rejected.
    http:Listener|http:ListenerError noStreams = new (9564, {http2MaxActiveStreams: 0});
    if noStreams is http:ListenerError {
        io:println(noStreams.message()); // @output http2MaxActiveStreams must be between 1 and 2147483647: 0
    }

    http:Listener|http:ListenerError hugeWindow = new (9565, {http2InitialWindowSize: 3000000000});
    if hugeWindow is http:ListenerError {
        io:println(hugeWindow.message()); // @output http2InitialWindowSize must be between 1 and 2147483647: 3000000000
    }

    http:Listener|http:ListenerError noPort = new (0);
    if noPort is http:ListenerError {
        io:println(noPort is http:GenericListenerError, " ", noPort.message()); // @output true Listener port is not defined!
    }
}
