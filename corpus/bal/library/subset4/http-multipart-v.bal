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
import ballerina/mime;
import ballerina/io;

isolated function hasPrefix(string value, string prefix) returns boolean {
    if value.length() < prefix.length() {
        return false;
    }
    return value.substring(0, prefix.length()) == prefix;
}

public function main() returns error? {
    mime:Entity part1 = new;
    part1.setText("hello world");
    mime:ContentDisposition disposition = check mime:getContentDispositionObject("form-data; name=\"field1\"");
    part1.setContentDisposition(disposition);

    mime:Entity part2 = new;
    part2.setJson({x: 1, y: 2});

    // ---- Request.setBodyParts / getBodyParts round-trip ----
    http:Request req = new;
    req.setBodyParts([part1, part2]);
    io:println(hasPrefix(req.getContentType(), "multipart/form-data")); // @output true

    mime:Entity[] reqParts = check req.getBodyParts();
    io:println(reqParts.length());                                        // @output 2
    io:println(check reqParts[0].getText());                              // @output hello world
    io:println(check reqParts[0].getHeader("content-disposition"));       // @output form-data; name=field1
    map<json> reqPartJson = <map<json>>check reqParts[1].getJson();
    io:println(reqPartJson["x"]); // @output 1
    io:println(reqPartJson["y"]); // @output 2

    // memoized second call
    mime:Entity[] reqPartsAgain = check req.getBodyParts();
    io:println(reqPartsAgain.length()); // @output 2

    // ---- Response.setBodyParts / getBodyParts round-trip ----
    http:Response res = new;
    res.setBodyParts([part1, part2]);
    mime:Entity[] resParts = check res.getBodyParts();
    io:println(resParts.length());          // @output 2
    io:println(check resParts[0].getText()); // @output hello world

    // ---- setPayload(mime:Entity[]) dispatch ----
    http:Request reqViaSetPayload = new;
    reqViaSetPayload.setPayload([part1, part2]);
    mime:Entity[] viaSetPayloadParts = check reqViaSetPayload.getBodyParts();
    io:println(viaSetPayloadParts.length()); // @output 2

    http:Response resViaSetPayload = new;
    resViaSetPayload.setPayload([part1, part2]);
    mime:Entity[] resViaSetPayloadParts = check resViaSetPayload.getBodyParts();
    io:println(resViaSetPayloadParts.length()); // @output 2

    // ---- explicit content-type override ----
    http:Request req2 = new;
    req2.setBodyParts([part1], "multipart/mixed");
    io:println(hasPrefix(req2.getContentType(), "multipart/mixed")); // @output true
}
