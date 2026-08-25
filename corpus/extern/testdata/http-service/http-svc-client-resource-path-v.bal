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

// The service echoes the request target back so the path and query string the
// client actually put on the wire can be asserted. Segments and query values use
// Ballerina's own toString rendering, and query components are percent-encoded by
// Go's net/url; both differ from jBallerina's wire format in the ways the http
// README records under "Notable Behavioural Changes".
service /p on new http:Listener(19223) {
    resource function get [string... rest](http:Request req) returns http:Response {
        return echoRawPath(req);
    }

    resource function post [string... rest](http:Request req) returns http:Response {
        return echoRawPath(req);
    }
}

// A second service mounted at the root base path, so an empty path segment
// list — the `client->/` form — has somewhere to dispatch to.
service / on new http:Listener(19224) {
    resource function get [string... rest](http:Request req) returns http:Response {
        return echoRawPath(req);
    }
}

function echoRawPath(http:Request req) returns http:Response {
    http:Response resp = new;
    resp.setHeader("x-raw-path", req.rawPath);
    return resp;
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19223", {});

    // A path segment of each PathParamType member. Note the numeric renderings:
    // a whole float keeps its ".0" and a decimal keeps its trailing zeros.
    http:Response ints = check c->/p/[0]/[42]/[-5];
    io:println(check ints.getHeader("x-raw-path")); // @output /p/0/42/-5

    http:Response bools = check c->/p/[true]/[false];
    io:println(check bools.getHeader("x-raw-path")); // @output /p/true/false

    http:Response floats = check c->/p/[1.0]/[1.5]/[-2.75];
    io:println(check floats.getHeader("x-raw-path")); // @output /p/1.0/1.5/-2.75

    // Outside [1e-3, 1e7) a float switches to scientific notation, rendered the
    // way lang.float:toString renders it (jBallerina's client would send
    // "1.0E10" here, because it formats path segments with Java's
    // Double.toString rather than with float:toString).
    http:Response sci = check c->/p/[1.0e10]/[1.0e-7];
    io:println(check sci.getHeader("x-raw-path")); // @output /p/1e10/1e-7

    // Non-finite floats render with their Ballerina names rather than as a
    // numeric literal.
    float nan = 0.0 / 0.0;
    float posInf = 1.0 / 0.0;
    float negInf = -1.0 / 0.0;
    http:Response nonFinite = check c->/p/[nan]/[posInf]/[negInf];
    io:println(check nonFinite.getHeader("x-raw-path")); // @output /p/NaN/Infinity/-Infinity

    http:Response nonFiniteQuery = check c->/p(bad = nan, pi = posInf);
    io:println(check nonFiniteQuery.getHeader("x-raw-path")); // @output /p?bad=NaN&pi=Infinity

    decimal whole = 1.0;
    decimal trailing = 2.500;
    http:Response decimals = check c->/p/[whole]/[trailing];
    io:println(check decimals.getHeader("x-raw-path")); // @output /p/1.0/2.500

    // A name segment and a computed string segment render identically.
    http:Response names = check c->/p/albums/tracks;
    io:println(check names.getHeader("x-raw-path")); // @output /p/albums/tracks

    string seg = "albums";
    http:Response computed = check c->/p/[seg]/["tracks"];
    io:println(check computed.getHeader("x-raw-path")); // @output /p/albums/tracks

    // Query parameters are appended after "?" and joined with "&", in the order
    // the named arguments were written.
    http:Response query = check c->/p/albums(tag = "rock", count = 3, exact = true);
    io:println(check query.getHeader("x-raw-path")); // @output /p/albums?tag=rock&count=3&exact=true

    // An array-valued query parameter becomes a single comma-joined pair rather
    // than a repeated key.
    http:Response arr = check c->/p/albums(tags = ["rock", "jazz", "folk"]);
    io:println(check arr.getHeader("x-raw-path")); // @output /p/albums?tags=rock,jazz,folk

    // Numeric query values use the same rendering as path segments.
    http:Response numQuery = check c->/p/albums(ratio = 1.5, big = 1.0e10, dec = trailing);
    io:println(check numQuery.getHeader("x-raw-path")); // @output /p/albums?ratio=1.5&big=1e10&dec=2.500

    // Query keys and values are escaped with net/url, so a space becomes "+" and
    // a value carrying a delimiter is encoded instead of splitting the parameter.
    http:Response spaced = check c->/p/albums(q = "a b");
    io:println(check spaced.getHeader("x-raw-path")); // @output /p/albums?q=a+b

    http:Response delim = check c->/p/albums(q = "a&b=c");
    io:println(check delim.getHeader("x-raw-path")); // @output /p/albums?q=a%26b%3Dc

    // Query parameters combine with a body-carrying accessor.
    http:Response withBody = check c->/p/albums.post("body", tag = "rock");
    io:println(check withBody.getHeader("x-raw-path")); // @output /p/albums?tag=rock

    // Headers and query parameters together.
    map<string|string[]> headers = {"x-custom": "sent"};
    http:Response both = check c->/p/albums(headers, tag = "rock");
    io:println(check both.getHeader("x-raw-path")); // @output /p/albums?tag=rock

    // An empty path segment list renders as the root path.
    http:Client rootClient = check new http:Client("http://localhost:19224", {});
    http:Response root = check rootClient->/;
    io:println(check root.getHeader("x-raw-path")); // @output /

    http:Response rootQuery = check rootClient->/(tag = "rock");
    io:println(check rootQuery.getHeader("x-raw-path")); // @output /?tag=rock

    return;
}
