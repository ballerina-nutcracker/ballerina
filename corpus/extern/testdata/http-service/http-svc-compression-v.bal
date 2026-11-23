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

final string big = createBig();

function createBig() returns string {
    string text = "";
    foreach int _ in 0 ..< 1000 {
        text += "0123456789";
    }
    return text;
}

listener http:Listener ep = new (19240);

service /d on ep {
    resource function get big() returns string => big;

    resource function get small() returns string => "small";

    resource function head small() returns string => "small";

    resource function get preset() returns http:Response {
        http:Response response = new;
        response.setTextPayload("preset");
        response.setHeader("Content-Encoding", "gzip");
        return response;
    }

    resource function get origin() returns http:Response {
        http:Response response = new;
        response.setTextPayload("origin");
        response.setHeader("Vary", "Origin");
        return response;
    }

    resource function get listed() returns http:Response {
        http:Response response = new;
        response.setTextPayload("listed");
        response.setHeader("Vary", "accept-encoding");
        return response;
    }

    resource function get unchanged() returns http:NotModified => {};

    resource function get none() returns http:NoContent => {};
}

@http:ServiceConfig {compression: {enable: http:COMPRESSION_ALWAYS}}
service /a on ep {
    resource function get small() returns string => "always";
}

@http:ServiceConfig {compression: {enable: http:COMPRESSION_NEVER}, chunking: http:CHUNKING_NEVER}
service /n on ep {
    resource function get big() returns string => big;
}

@http:ServiceConfig {compression: {contentTypes: ["application/json"]}, chunking: http:CHUNKING_ALWAYS}
service /c on ep {
    resource function get text() returns string => "text";

    resource function get data() returns json => {a: big};
}

function showVary(http:Client c, string path, string? acceptEncoding = "gzip", boolean head = false) returns error? {
    map<string> headers = acceptEncoding is string ? {"Accept-Encoding": acceptEncoding} : {};
    http:Response response;
    if head {
        response = check c->head(path, headers);
    } else {
        response = check c->get(path, headers);
    }
    string[]|error vary = response.getHeaders("vary");
    io:println(path, " ", response.statusCode, " ", vary is error ? "-" : vary.toString());
}

function show(http:Client c, string path, string? acceptEncoding = ()) returns error? {
    map<string> headers = acceptEncoding is string ? {"Accept-Encoding": acceptEncoding} : {};
    http:Response response = check c->get(path, headers);
    string|error length = response.getHeader("content-length");
    string body = check response.getTextPayload();
    io:println(response.statusCode, " ", body.length(), " ", length is string ? "sized" : "unsized");
}

public function testMain() returns error? {
    http:Client c = check new ("http://127.0.0.1:19240");
    http:Client h1 = check new ("http://127.0.0.1:19240", {httpVersion: http:HTTP_1_1});

    check show(c, "/d/small", "gzip");
    // @output 200 5 unsized
    check show(c, "/d/big", "deflate");
    // @output 200 10000 unsized
    check show(c, "/d/big", "br");
    // @output 200 10000 unsized
    check show(c, "/d/preset");
    // @output 200 6 unsized
    check show(c, "/a/small");
    // @output 200 6 unsized
    check show(h1, "/n/big", "gzip");
    // @output 200 10000 sized
    check show(h1, "/c/text", "gzip");
    // @output 200 4 unsized
    check show(h1, "/c/data", "gzip");
    // @output 200 10008 unsized
    check show(h1, "/d/small");
    // @output 200 5 sized

    // jBallerina sends no Vary for a negotiated coding: https://github.com/ballerina-platform/ballerina-library/issues/9898
    check showVary(c, "/d/small");
    // @output /d/small 200 ["Accept-Encoding"]
    check showVary(c, "/d/small", "identity");
    // @output /d/small 200 ["Accept-Encoding"]
    check showVary(c, "/a/small");
    // @output /a/small 200 ["Accept-Encoding"]
    check showVary(c, "/d/small", head = true);
    // @output /d/small 200 ["Accept-Encoding"]
    check showVary(c, "/d/origin");
    // @output /d/origin 200 ["Origin","Accept-Encoding"]
    check showVary(c, "/d/listed");
    // @output /d/listed 200 ["accept-encoding"]
    check showVary(c, "/d/unchanged");
    // @output /d/unchanged 304 ["Accept-Encoding"]
    check showVary(c, "/d/none");
    // @output /d/none 204 -
    check showVary(c, "/n/big");
    // @output /n/big 200 -
    check showVary(c, "/d/preset");
    // @output /d/preset 200 -
    check showVary(c, "/c/text");
    // @output /c/text 200 -
}
