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
import ballerina/mime;

public function main() returns error? {
    // ---- Request.setPayload ----
    http:Request reqText = new;
    reqText.setPayload("plain text");
    io:println(check reqText.getTextPayload()); // @output plain text

    http:Request reqBytes = new;
    reqBytes.setPayload("bin".toBytes());
    byte[] reqBytesResult = check reqBytes.getBinaryPayload();
    io:println(reqBytesResult.length()); // @output 3

    http:Request reqJson = new;
    reqJson.setPayload({name: "Alice", age: 30});
    json reqJsonResult = check reqJson.getJsonPayload();
    map<json> reqJsonMap = <map<json>>reqJsonResult;
    io:println(reqJsonMap["name"]); // @output Alice
    io:println(reqJsonMap["age"]);  // @output 30

    http:Request reqXml = new;
    reqXml.setPayload(xml `<greeting>hello</greeting>`);
    io:println(reqXml.getContentType());      // @output application/xml
    io:println(check reqXml.getXmlPayload()); // @output <greeting>hello</greeting>

    http:Request reqXmlCustomType = new;
    reqXmlCustomType.setPayload(xml `<a/>`, "text/xml");
    io:println(reqXmlCustomType.getContentType()); // @output text/xml

    // ---- Response.setPayload ----
    http:Response resText = new;
    resText.setPayload("resp text");
    io:println(check resText.getTextPayload()); // @output resp text

    http:Response resBytes = new;
    resBytes.setPayload("xyz".toBytes());
    byte[] resBytesResult = check resBytes.getBinaryPayload();
    io:println(resBytesResult.length()); // @output 3

    http:Response resJson = new;
    json resPayload = [1, 2, 3];
    resJson.setPayload(resPayload);
    json resJsonResult = check resJson.getJsonPayload();
    io:println(resJsonResult); // @output [1,2,3]

    http:Response resXml = new;
    resXml.setPayload(xml `<item id="1"/>`);
    io:println(resXml.getContentType());      // @output application/xml
    io:println(check resXml.getXmlPayload()); // @output <item id="1"/>

    // ---- XML body part in a multipart payload ----
    mime:Entity xmlPart = new;
    xmlPart.setXml(xml `<note>multipart</note>`);
    http:Request reqMultipartXml = new;
    reqMultipartXml.setPayload([xmlPart]);
    mime:Entity[] xmlParts = check reqMultipartXml.getBodyParts();
    io:println(check xmlParts[0].getXml());                // @output <note>multipart</note>
    io:println(check xmlParts[0].getHeader("content-type")); // @output application/xml
}
