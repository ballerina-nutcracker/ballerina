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

listener http:Listener ep = new (19241);

@http:ServiceConfig {
    cors: {
        allowOrigins: ["http://a.com", "http://b.com"],
        allowCredentials: true,
        allowHeaders: ["X-A", "X-C"],
        exposeHeaders: ["X-E", "X-F"],
        maxAge: 60
    }
}
service /c on ep {
    resource function get r() returns string => "r";

    resource function post r() returns string => "p";

    resource function get err() returns error => error("bad");

    resource function get preset() returns http:Response {
        http:Response response = new;
        response.setHeader("Access-Control-Allow-Origin", "http://preset.com");
        response.setTextPayload("preset");
        return response;
    }

    resource function get vary/[string value]() returns http:Response {
        http:Response response = new;
        response.setHeader("Vary", value);
        response.setTextPayload("vary");
        return response;
    }

    resource function 'default d() returns string => "d";
}

@http:ServiceConfig {cors: {allowOrigins: ["*"]}}
service /star on ep {
    resource function get r() returns string => "r";

    resource function get rest() returns string => "rest";

    resource function post rest/[string... path]() returns string => "rest";
}

// Credentials are allowed only for listed origins (ballerina-library#9882).
@http:ServiceConfig {cors: {allowCredentials: true}}
service /credentials on ep {
    resource function get r() returns string => "r";
}

@http:ServiceConfig {cors: {allowOrigins: ["http://a.com"], maxAge: 1.5}}
service /own on ep {
    resource function get r() returns http:MethodNotAllowed => {headers: {"Allow": "POST"}};

    resource function purge r() returns string => "purged";

    resource function 'default d() returns string => "d";
}

service /onlyOptions on ep {
    resource function options a() returns string => "a";
}

service /plain on ep {
    resource function get r() returns string => "r";

    resource function post r() returns string => "r";
}

service "/my albums" on ep {
    resource function get r() returns string => "r";
}

const string[] CORS_HEADERS = ["allow", "vary", "access-control-allow-origin", "access-control-allow-credentials",
    "access-control-allow-methods", "access-control-allow-headers", "access-control-max-age",
    "access-control-expose-headers"];

function show(http:Client c, string method, string path, map<string> headers = {}) returns error? {
    http:Response response = check c->execute(method, path, (), headers);
    string line = response.statusCode.toString();
    foreach string name in CORS_HEADERS {
        string|error value = response.getHeader(name);
        if value is string {
            line += string ` ${name}=[${value}]`;
        }
    }
    io:println(line);
}

function showVary(http:Client c, string path) returns error? {
    http:Response response = check c->get(path, {Origin: "http://a.com"});
    io:println(check response.getHeaders("Vary"));
}

public function testMain() returns error? {
    http:Client c = check new ("http://127.0.0.1:19241");
    map<string> fromA = {Origin: "http://a.com"};

    // jBallerina sends no Vary: Origin with the CORS headers (ballerina-library#9564).
    check show(c, "GET", "/c/r", fromA);
    // @output 200 vary=[Origin] access-control-allow-origin=[http://a.com] access-control-allow-credentials=[true] access-control-expose-headers=[X-E, X-F]
    check show(c, "GET", "/c/r", {Origin: "http://a.com http://b.com"});
    // @output 200 vary=[Origin] access-control-allow-origin=[http://a.com http://b.com] access-control-allow-credentials=[true] access-control-expose-headers=[X-E, X-F]
    check show(c, "GET", "/c/r", {Origin: "http://x.com"});
    // @output 200 vary=[Origin]
    check show(c, "GET", "/c/r");
    // @output 200 vary=[Origin]
    check show(c, "GET", "/c/err", fromA);
    // @output 500 vary=[Origin] access-control-allow-origin=[http://a.com] access-control-allow-credentials=[true] access-control-expose-headers=[X-E, X-F]
    check show(c, "GET", "/c/preset", fromA);
    // @output 200 vary=[Origin] access-control-allow-origin=[http://a.com] access-control-allow-credentials=[true] access-control-expose-headers=[X-E, X-F]
    // jBallerina sends no Vary: Accept-Encoding, here and on the 404, 405 and /plain lines below (ballerina-library#9898).
    check showVary(c, "/c/vary/Origin");
    // @output ["Origin","Accept-Encoding"]
    check showVary(c, "/c/vary/*");
    // @output ["*"]
    check show(c, "DELETE", "/c/r", fromA);
    // @output 405 vary=[Accept-Encoding]

    // jBallerina rejects request headers listed with spaces after the commas (ballerina-library#9566).
    check show(c, "OPTIONS", "/c/r", {
        Origin: "http://a.com",
        "Access-Control-Request-Method": "POST",
        "Access-Control-Request-Headers": "x-a, X-C"
    });
    // @output 204 vary=[Origin] access-control-allow-origin=[http://a.com] access-control-allow-credentials=[true] access-control-allow-methods=[POST] access-control-allow-headers=[x-a, X-C] access-control-max-age=[60]
    check show(c, "OPTIONS", "/c/r", {Origin: "http://a.com", "Access-Control-Request-Method": "HEAD"});
    // @output 204 vary=[Origin] access-control-allow-origin=[http://a.com] access-control-allow-credentials=[true] access-control-allow-methods=[HEAD] access-control-max-age=[60]
    check show(c, "OPTIONS", "/c/r", {Origin: "http://a.com", "Access-Control-Request-Method": "DELETE"});
    // @output 204 allow=[GET, POST, OPTIONS]
    check show(c, "OPTIONS", "/c/r", {
        Origin: "http://a.com",
        "Access-Control-Request-Method": "POST",
        "Access-Control-Request-Headers": "X-B"
    });
    // @output 204 allow=[GET, POST, OPTIONS]
    check show(c, "OPTIONS", "/c/r", {
        Origin: "http://a.com",
        "Access-Control-Request-Method": "POST",
        "Access-Control-Request-Headers": "X-A,"
    });
    // @output 204 vary=[Origin] access-control-allow-origin=[http://a.com] access-control-allow-credentials=[true] access-control-allow-methods=[POST] access-control-allow-headers=[X-A] access-control-max-age=[60]
    check show(c, "OPTIONS", "/c/r", {Origin: "http://x.com", "Access-Control-Request-Method": "GET"});
    // @output 204 allow=[GET, POST, OPTIONS]
    check show(c, "OPTIONS", "/c/r");
    // @output 204 allow=[GET, POST, OPTIONS]
    check show(c, "OPTIONS", "/c/d", fromA);
    // @output 200 vary=[Origin] access-control-allow-origin=[http://a.com] access-control-allow-credentials=[true] access-control-expose-headers=[X-E, X-F]
    check show(c, "OPTIONS", "/c");
    // @output 204 allow=[GET, HEAD, POST, DELETE, PATCH, PUT, OPTIONS]
    check show(c, "OPTIONS", "/c/missing");
    // @output 404 vary=[Accept-Encoding]
    check show(c, "OPTIONS", "/c?x=1");
    // @output 404 vary=[Accept-Encoding]
    check show(c, "OPTIONS", "/c/");
    // @output 204 allow=[GET, HEAD, POST, DELETE, PATCH, PUT, OPTIONS]

    check show(c, "GET", "/star/r", fromA);
    // @output 200 vary=[Origin] access-control-allow-origin=[http://a.com]
    // jBallerina sends Access-Control-Max-Age: -1 when maxAge is not set (ballerina-library#9565).
    check show(c, "OPTIONS", "/star/r", {Origin: "http://a.com", "Access-Control-Request-Method": "GET"});
    // @output 204 vary=[Origin] access-control-allow-origin=[http://a.com] access-control-allow-methods=[GET]
    // jBallerina matches a rest path parameter to no segments neither in dispatch nor in Allow (ballerina-library#9560, #9655).
    check show(c, "OPTIONS", "/star/rest");
    // @output 204 allow=[GET, POST, OPTIONS]
    check show(c, "OPTIONS", "/star/rest", {Origin: "http://a.com", "Access-Control-Request-Method": "POST"});
    // @output 204 vary=[Origin] access-control-allow-origin=[http://a.com] access-control-allow-methods=[POST]
    check show(c, "OPTIONS", "/star");
    // @output 204 allow=[GET, POST, OPTIONS]

    check show(c, "GET", "/credentials/r", {Origin: "http://x.com"});
    // @output 200 vary=[Origin] access-control-allow-origin=[http://x.com]
    // jBallerina removes the resource's Allow header (ballerina-library#9883).
    check show(c, "GET", "/own/r", fromA);
    // @output 405 allow=[POST] vary=[Origin] access-control-allow-origin=[http://a.com]
    check show(c, "OPTIONS", "/own/r", {Origin: "http://a.com", "Access-Control-Request-Method": "PURGE"});
    // @output 204 vary=[Origin] access-control-allow-origin=[http://a.com] access-control-allow-methods=[PURGE] access-control-max-age=[2]
    check show(c, "OPTIONS", "/own");
    // @output 204 allow=[GET, HEAD, POST, DELETE, PATCH, PUT, PURGE, OPTIONS]
    check show(c, "OPTIONS", "/onlyOptions");
    // @output 204 allow=[OPTIONS]
    check show(c, "OPTIONS", "/my%20albums");
    // @output 204 allow=[GET, OPTIONS]
    check show(c, "OPTIONS", "/pl%61in");
    // @output 404 vary=[Accept-Encoding]

    check show(c, "GET", "/plain/r", fromA);
    // @output 200 vary=[Accept-Encoding]
    check show(c, "OPTIONS", "/plain/r", {Origin: "http://a.com", "Access-Control-Request-Method": "GET"});
    // @output 204 allow=[GET, POST, OPTIONS]
}
