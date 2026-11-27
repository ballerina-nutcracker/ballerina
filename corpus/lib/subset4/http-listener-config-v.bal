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
    http:ListenerConfiguration config = {};
    io:println(config.host, " ", config.timeout, " ", config.gracefulStopTimeout); // @output 0.0.0.0 60 0
    io:println(config.http1Settings); // @output {"keepAlive":"AUTO","maxPipelinedRequests":10}
    // @output {"soBackLog":100,"connectTimeOut":15,"receiveBufferSize":1048576,"sendBufferSize":1048576,"tcpNoDelay":true,"socketReuse":true,"keepAlive":false}
    io:println(config.socketConfig);
    // @output 65535 100 300 30
    io:println(config.http2InitialWindowSize, " ", config.http2MaxActiveStreams, " ", config.minIdleTimeInStaleState, " ",
        config.timeBetweenStaleEviction);
    io:println(http:DEFAULT_LISTENER_TIMEOUT, " ", http:DEFAULT_GRACEFULSTOP_TIMEOUT); // @output 60 0

    http:Listener tuned = check new (19581, {
        http1Settings: {keepAlive: http:KEEPALIVE_NEVER, maxPipelinedRequests: 2},
        timeout: 0,
        gracefulStopTimeout: 5,
        socketConfig: {receiveBufferSize: 4096, sendBufferSize: 4096, tcpNoDelay: false, keepAlive: true, soBackLog: 10},
        http2InitialWindowSize: 70000,
        http2MaxActiveStreams: 7
    });
    io:println(tuned.getPort()); // @output 19581
}
