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
import ballerina/mime;

public function main() {
    mime:Entity entity = new ();
    entity.setHeader("Content-Type", "application/json");

    string|mime:HeaderNotFoundError ct = entity.getHeader("content-type");
    if ct is string {
        io:println(ct);
    }

    io:println(entity.hasHeader("content-type"));
    io:println(entity.hasHeader("accept"));

    entity.addHeader("Accept", "text/html");
    entity.addHeader("Accept", "application/json");
    string[]|mime:HeaderNotFoundError accepts = entity.getHeaders("accept");
    if accepts is string[] {
        io:println(accepts.length());
    }

    string[] names = entity.getHeaderNames();
    io:println(names.length());

    entity.removeHeader("Accept");
    io:println(entity.hasHeader("accept"));

    entity.removeAllHeaders();
    io:println(entity.hasHeader("content-type"));
}
// @output application/json
// @output true
// @output false
// @output 2
// @output 2
// @output false
// @output false
