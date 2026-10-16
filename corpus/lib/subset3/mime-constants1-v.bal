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
    io:println(mime:APPLICATION_JSON);
    io:println(mime:TEXT_PLAIN);
    io:println(mime:APPLICATION_OCTET_STREAM);
    io:println(mime:CONTENT_TYPE);
    io:println(mime:CONTENT_LENGTH);
    io:println(mime:DEFAULT_CHARSET);
    io:println(mime:MULTIPART_FORM_DATA);
    io:println(mime:IMAGE_JPEG);
}
// @output application/json
// @output text/plain
// @output application/octet-stream
// @output content-type
// @output content-length
// @output UTF-8
// @output multipart/form-data
// @output image/jpeg
