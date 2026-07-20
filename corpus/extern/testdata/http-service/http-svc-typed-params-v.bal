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

// Exercises the runtime path dispatcher's type coercion across boolean,
// decimal, and string path parameters.
service /api on new http:Listener(19196) {
    resource function get flag/[boolean b]() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload(string `flag=${b}`);
        return resp;
    }

    resource function get price/[decimal d]() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload(string `price=${d}`);
        return resp;
    }

    resource function get echo/[string s]() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload(s);
        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19196", {});
    http:Response flag = check c->get("/api/flag/true");
    io:println(flag.getTextPayload()); // @output flag=true
    http:Response price = check c->get("/api/price/9.99");
    io:println(price.getTextPayload()); // @output price=9.99
    http:Response echo = check c->get("/api/echo/hello");
    io:println(echo.getTextPayload()); // @output hello
}
