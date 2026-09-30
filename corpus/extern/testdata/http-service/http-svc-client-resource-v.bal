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

type Album record {|
    int id;
    string tag;
|};

listener http:Listener resourceListener = new (19225);

// Each accessor echoes back the method the listener actually dispatched on, as a
// header and as the text payload, so a client resource method bound to the wrong
// native implementation would surface as a mismatched verb rather than passing
// silently. The extern lookup key of a resource method embeds its declaration
// index within the class, which makes that pairing worth asserting for every
// accessor.
service /verb on resourceListener {
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
    resp.setTextPayload(req.method);
    return resp;
}

service /albums on resourceListener {
    resource function get [int id](http:Request req) returns Album {
        return {id, tag: req.getQueryParamValue("tag") ?: ""};
    }

    resource function post [int id](http:Request req) returns Album|error {
        Album album = check (check req.getJsonPayload()).cloneWithType();
        return {id, tag: album.tag};
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19225", {});

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

    // Every accessor except `head` binds the payload to the expected type, which
    // also asserts each one reads `targetType` from its own argument position.
    string getBody = check c->/verb/albums/[1];
    io:println(getBody); // @output GET

    string postBody = check c->/verb/albums.post("body");
    io:println(postBody); // @output POST

    string putBody = check c->/verb/albums.put("body");
    io:println(putBody); // @output PUT

    string patchBody = check c->/verb/albums.patch("body");
    io:println(patchBody); // @output PATCH

    string deleteBody = check c->/verb/albums.delete();
    io:println(deleteBody); // @output DELETE

    string optionsBody = check c->/verb/albums.options();
    io:println(optionsBody); // @output OPTIONS

    // A JSON payload binds to a record, with query parameters and a request body
    // travelling alongside the inferred `targetType`.
    Album fetched = check c->/albums/[7](tag = "rock");
    io:println(fetched.id, " ", fetched.tag); // @output 7 rock

    Album created = check c->/albums/[8].post({id: 0, tag: "jazz"});
    io:println(created.id, " ", created.tag); // @output 8 jazz

    // A payload that does not fit the target type is a binding error.
    int|error mismatched = c->/albums/[9](tag = "folk");
    io:println(mismatched is error); // @output true

    return;
}
