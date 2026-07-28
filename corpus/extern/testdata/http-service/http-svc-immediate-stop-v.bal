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

listener http:Listener ep = new (19210);

service /svc on ep {
    resource function get ping() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload("pong");
        return resp;
    }

    // immediateStop force-closes the listener synchronously, so any request
    // issued after this resource runs is refused.
    resource function get stop() returns error? {
        check ep.immediateStop();
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19210", {});

    // The service is up: a normal request succeeds.
    http:Response r = check c->get("/svc/ping");
    io:println(r.statusCode); // @output 200

    // Trigger the stop; Close() severs the in-flight connection before the
    // handler's response reaches the client, so this call errors.
    http:Response|error stopResp = c->get("/svc/stop");
    io:println(stopResp is error); // @output true

    // The listener is now closed, so a fresh request is connection-refused.
    http:Response|error afterStop = c->get("/svc/ping");
    io:println(afterStop is error); // @output true
}
