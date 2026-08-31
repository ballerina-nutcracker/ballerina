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

// Each accessor echoes back the method the listener actually dispatched on, so a
// client resource method bound to the wrong native implementation would surface
// as a mismatched verb rather than passing silently. The extern lookup key of a
// resource method embeds its declaration index within the class, which makes
// that pairing worth asserting for every accessor.
service /verb on new http:Listener(19222) {
    resource function get [string... rest](http:Request req) returns http:Response {
        return echoMethod(req);
    }

    resource function post [string... rest](http:Request req) returns http:Response {
        return echoMethod(req);
    }

    resource function put [string... rest](http:Request req) returns http:Response {
        return echoMethod(req);
    }

    resource function patch [string... rest](http:Request req) returns http:Response {
        return echoMethod(req);
    }

    resource function delete [string... rest](http:Request req) returns http:Response {
        return echoMethod(req);
    }

    resource function head [string... rest](http:Request req) returns http:Response {
        return echoMethod(req);
    }

    resource function options [string... rest](http:Request req) returns http:Response {
        return echoMethod(req);
    }
}

function echoMethod(http:Request req) returns http:Response {
    http:Response resp = new;
    resp.setHeader("x-method", req.method);
    resp.setHeader("x-raw-path", req.rawPath);
    return resp;
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19222", {});

    // `get` is the accessor used when the call site names none.
    http:Response getResp = check c->/verb/albums/[1];
    io:println(getResp.statusCode); // @output 200
    io:println(check getResp.getHeader("x-method")); // @output GET
    io:println(check getResp.getHeader("x-raw-path")); // @output /verb/albums/1

    http:Response postResp = check c->/verb/albums.post("body");
    io:println(check postResp.getHeader("x-method")); // @output POST

    http:Response putResp = check c->/verb/albums.put("body");
    io:println(check putResp.getHeader("x-method")); // @output PUT

    http:Response patchResp = check c->/verb/albums.patch("body");
    io:println(check patchResp.getHeader("x-method")); // @output PATCH

    // `delete` takes an optional message, so both forms must dispatch.
    http:Response deleteResp = check c->/verb/albums.delete();
    io:println(check deleteResp.getHeader("x-method")); // @output DELETE

    http:Response deleteBodyResp = check c->/verb/albums.delete("body");
    io:println(check deleteBodyResp.getHeader("x-method")); // @output DELETE

    http:Response headResp = check c->/verb/albums.head();
    io:println(check headResp.getHeader("x-method")); // @output HEAD

    http:Response optionsResp = check c->/verb/albums.options();
    io:println(check optionsResp.getHeader("x-method")); // @output OPTIONS

    // Request headers travel with a resource-method call, and the media type
    // override reaches the server as the Content-Type.
    map<string|string[]> headers = {"x-custom": "sent"};
    http:Response hdrResp = check c->/verb/albums(headers);
    io:println(check hdrResp.getHeader("x-method")); // @output GET

    http:Response typedResp = check c->/verb/albums.post("body", mediaType = "text/plain");
    io:println(check typedResp.getHeader("x-method")); // @output POST

    return;
}
