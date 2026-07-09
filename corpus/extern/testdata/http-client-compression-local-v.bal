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
    http:Client c = check new http:Client("http://testserver", {});

    // gzip-encoded response -> decompressResponseBody wraps the body in a
    // gzipReadCloser and strips the Content-Encoding header before the payload
    // is read transparently as text.
    http:Response gz = check c->get("/gzip");
    io:println(gz.getTextPayload()); // @output hello gzipped world

    // deflate-encoded response -> decompressResponseBody + deflateReadCloser.
    http:Response df = check c->get("/deflate");
    io:println(df.getTextPayload()); // @output hello deflated world

    // A response advertising gzip but carrying a malformed body must surface a
    // read error rather than silently returning the raw compressed bytes.
    http:Response bad = check c->get("/badgzip");
    string|error payload = bad.getTextPayload();
    io:println(payload is error); // @output true
    return;
}
