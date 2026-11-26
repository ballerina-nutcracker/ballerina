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

// The annotation has no effect yet (#1355), so the listener is never started.
service class Albums {
    *http:Service;

    resource function get albums() returns @http:Cache string => "albums";

    resource function get album() returns @http:Cache {maxAge: 10, isPrivate: true, noCacheFields: ["a"]} http:Ok => {body: "album"};
}

public function main() returns error? {
    http:HttpCacheConfig defaults = {};
    io:println(defaults); // @output {"mustRevalidate":true,"noCache":false,"noStore":false,"noTransform":false,"isPrivate":false,"proxyRevalidate":false,"maxAge":3600,"sMaxAge":-1,"noCacheFields":[],"privateFields":[],"setETag":true,"setLastModified":true}

    http:Listener ln = check new (19612);
    check ln.attach(new Albums(), "/music");
    io:println("attached"); // @output attached
}
