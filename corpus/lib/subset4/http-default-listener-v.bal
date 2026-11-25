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

service class Greeter {
    *http:Service;

    resource function get hello() returns string => "hello";
}

public function main() returns error? {
    http:Listener first = check http:getDefaultListener();
    http:Listener second = check http:getDefaultListener();
    io:println(first === second); // @output true
    io:println(first.getPort()); // @output 9090

    io:println(http:defaultListenerPort); // @output 9090
    io:println(http:defaultListenerConfig.host); // @output 0.0.0.0
    io:println(http:defaultListenerConfig.timeout); // @output 60
    io:println(http:defaultListenerConfig.httpVersion); // @output 2.0

    // Never started, so the test does not bind port 9090.
    check first.attach(new Greeter(), "/greeter");
    io:println("attached"); // @output attached

    http:Listener explicit = check new (19580);
    io:println(explicit.getPort()); // @output 19580
}
