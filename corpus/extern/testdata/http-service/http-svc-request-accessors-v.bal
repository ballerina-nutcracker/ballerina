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

service /acc on new http:Listener(19212) {
    resource function get info(http:Request req) returns http:Response {
        http:Response resp = new;

        // rawPath must carry the original request-target, including the
        // query string, not just the decoded path.
        resp.setHeader("x-raw-path", req.rawPath);

        resp.setHeader("x-content-type", req.getContentType());

        string[] names = req.getHeaderNames();
        boolean hasContentType = false;
        foreach string name in names {
            if name == "content-type" {
                hasContentType = true;
            }
        }
        if hasContentType {
            resp.setHeader("x-has-content-type-name", "true");
        } else {
            resp.setHeader("x-has-content-type-name", "false");
        }

        string[]? vals = req.getQueryParamValues("tag");
        if vals is string[] {
            resp.setHeader("x-tag-1", vals[0]);
            resp.setHeader("x-tag-2", vals[1]);
        }

        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19212", {});
    map<string|string[]> headers = {"content-type": "text/plain"};
    http:Response r = check c->get("/acc/info?tag=a&tag=b", headers);
    io:println(r.statusCode); // @output 200
    io:println(check r.getHeader("x-raw-path")); // @output /acc/info?tag=a&tag=b
    io:println(check r.getHeader("x-content-type")); // @output text/plain
    io:println(check r.getHeader("x-has-content-type-name")); // @output true
    io:println(check r.getHeader("x-tag-1")); // @output a
    io:println(check r.getHeader("x-tag-2")); // @output b
}
