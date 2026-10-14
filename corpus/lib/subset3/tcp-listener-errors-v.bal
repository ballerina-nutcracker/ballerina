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

import ballerina/io;
import ballerina/tcp;

service class Svc {
    *tcp:Service;
}

public function main() {
    tcp:Client|tcp:Error c = new ("127.0.0.1", 1, secureSocket = {cert: "/nonexistent/ca.pem"});
    io:println(c is tcp:Error ? c.message() : "connected"); // @output File does not contain valid certificates: /nonexistent/ca.pem

    tcp:Listener|tcp:Error l = new (21901);
    if l is tcp:Error {
        return;
    }
    error? g = l.gracefulStop();
    io:println(g is error ? g.message() : "ok"); // @output Unable to initialize the tcp listener.
    error? r1 = l.attach(new Svc());
    error? r2 = l.attach(new Svc());
    io:println(r1 is (), " ", r2 is error); // @output true true
    error? d = l.detach(new Svc());
    io:println(d is error); // @output true
}
