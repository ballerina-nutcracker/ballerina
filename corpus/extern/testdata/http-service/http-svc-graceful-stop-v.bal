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

listener http:Listener ep = new (19199);

// A resource calling ep.gracefulStop() on itself must not self-deadlock: the
// extern runs inline on this resource's own goroutine, so it must return
// immediately and let the shutdown drain in the background instead of
// blocking until the connection (this very request) goes idle.
service /svc on ep {
    resource function get stop() returns error? {
        check ep.gracefulStop();
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19199", {});
    http:Response r = check c->get("/svc/stop");
    io:println(r.statusCode); // @output 202
}
