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
    // A fresh response has no Content-Type, so the payload's default applies.
    http:Response res = new;
    res.setXmlPayload(xml `<a><b>1</b></a>`);
    io:println(res.getContentType()); // @output application/xml
    xml payload = check res.getXmlPayload();
    io:println(payload); // @output <a><b>1</b></a>
    io:println(payload is xml:Element); // @output true

    http:Response overridden = new;
    overridden.setXmlPayload(xml `<a/>`, "application/atom+xml");
    io:println(overridden.getContentType()); // @output application/atom+xml

    // An already-set Content-Type is preserved when no override is passed.
    http:Response preset = new;
    check preset.setContentType("application/soap+xml");
    preset.setXmlPayload(xml `<a/>`);
    io:println(preset.getContentType()); // @output application/soap+xml

    // An empty body is not bindable to xml.
    http:Response empty = new;
    xml|error noContent = empty.getXmlPayload();
    if noContent is error {
        io:println(noContent.message()); // @output No content
    }

    // The same behaviours on a request.
    http:Request req = new;
    req.setXmlPayload(xml `<p>hi</p>`);
    io:println(req.getContentType()); // @output application/xml
    xml reqPayload = check req.getXmlPayload();
    io:println(reqPayload); // @output <p>hi</p>

    http:Request malformed = new;
    malformed.setTextPayload("<a><b></a>", "application/xml");
    xml|error broken = malformed.getXmlPayload();
    if broken is error {
        io:println(broken.message()); // @output Error occurred while retrieving the xml payload from the request
    }

    // An XML declaration is kept as a processing instruction, matching jBallerina, so a
    // declaration-prefixed document is a sequence rather than a single element.
    http:Request declared = new;
    declared.setTextPayload("<?xml version=\"1.0\"?><a/>", "application/xml");
    xml withDecl = check declared.getXmlPayload();
    io:println(withDecl); // @output <?xml version="1.0"?><a/>
    io:println(withDecl is xml:Element); // @output false

    // A non-element payload keeps its own xml subtype.
    http:Request mixed = new;
    mixed.setXmlPayload(xml `<!--note-->`);
    xml comment = check mixed.getXmlPayload();
    io:println(comment is xml:Comment); // @output true

    return;
}
