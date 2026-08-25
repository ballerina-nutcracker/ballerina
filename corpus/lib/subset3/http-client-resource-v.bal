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
    http:Client c = check new ("https://example.com");

    // Every accessor is reachable through the client resource access syntax and
    // returns the raw response. `get` is the accessor when the call names none.
    http:Response getResp = check c->/albums/[1];
    io:println(getResp.statusCode); // @output 200
    io:println(check getResp.getTextPayload()); // @output test body

    http:Response postResp = check c->/albums.post("req");
    io:println(check postResp.getTextPayload()); // @output test body

    http:Response putResp = check c->/albums/[1].put("req");
    io:println(check putResp.getTextPayload()); // @output test body

    http:Response patchResp = check c->/albums/[1].patch("req");
    io:println(check patchResp.getTextPayload()); // @output test body

    // `delete` takes an optional message, so both forms are valid.
    http:Response deleteResp = check c->/albums/[1].delete();
    io:println(check deleteResp.getTextPayload()); // @output test body

    http:Response deleteBodyResp = check c->/albums/[1].delete("req");
    io:println(check deleteBodyResp.getTextPayload()); // @output test body

    http:Response headResp = check c->/albums.head();
    io:println(headResp.statusCode); // @output 200

    http:Response optionsResp = check c->/albums.options();
    io:println(check optionsResp.getTextPayload()); // @output test body

    // A path may be the bare root, a chain of name segments, or a mix of name
    // and computed segments covering every `PathParamType` member.
    http:Response rootResp = check c->/;
    io:println(rootResp.statusCode); // @output 200

    http:Response nameResp = check c->/albums/tracks/comments;
    io:println(nameResp.statusCode); // @output 200

    decimal price = 2.5;
    http:Response mixedResp = check c->/albums/[1]/[true]/[1.5]/[price]/["name"];
    io:println(mixedResp.statusCode); // @output 200

    // Headers and query parameters are both accepted, separately and together.
    // Query parameters are named arguments gathered by `*QueryParams`.
    http:Response hdrResp = check c->/albums({"X-Custom": "value"});
    io:println(hdrResp.statusCode); // @output 200

    http:Response queryResp = check c->/albums(tag = "rock", count = 3, exact = true);
    io:println(queryResp.statusCode); // @output 200

    http:Response arrQueryResp = check c->/albums(tags = ["rock", "jazz"]);
    io:println(arrQueryResp.statusCode); // @output 200

    http:Response bothResp = check c->/albums({"X-Custom": "value"}, tag = "rock");
    io:println(bothResp.statusCode); // @output 200

    // Query parameters also combine with a body-carrying accessor, alongside an
    // explicit media type override.
    http:Response postQueryResp = check c->/albums.post("req", mediaType = "text/plain", tag = "rock");
    io:println(check postQueryResp.getTextPayload()); // @output test body

    return;
}
