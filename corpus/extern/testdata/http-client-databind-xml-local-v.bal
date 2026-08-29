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

service /db on new http:Listener(19222) {
    resource function get elem() returns xml {
        return xml `<a><b>1</b></a>`;
    }

    // Go's decoder reports an XML declaration as a processing instruction, and jBallerina
    // keeps it in the parsed value too, so the document is a two-item sequence.
    resource function get decl() returns http:Response {
        http:Response r = new;
        r.setTextPayload("<?xml version=\"1.0\"?><a/>", "application/xml");
        return r;
    }

    resource function get textXml() returns http:Response {
        http:Response r = new;
        r.setTextPayload("<a>1</a>", "text/xml");
        return r;
    }

    resource function get suffixXml() returns http:Response {
        http:Response r = new;
        r.setTextPayload("<feed/>", "application/atom+xml");
        return r;
    }

    resource function get seqXml() returns http:Response {
        http:Response r = new;
        r.setTextPayload("<a/><b/>", "application/xml");
        return r;
    }

    resource function get emptyXml() returns http:Response {
        http:Response r = new;
        r.setTextPayload("", "application/xml");
        return r;
    }

    resource function get brokenXml() returns http:Response {
        http:Response r = new;
        r.setTextPayload("<a><b></a>", "application/xml");
        return r;
    }

    // Proves getXmlPayload works on a natively built inbound request, which dispatches
    // through requestMethodKeyCache rather than the class VTable.
    resource function post echo(http:Request req) returns xml|error {
        return req.getXmlPayload();
    }

    resource function post ctype(http:Request req) returns string {
        return req.getContentType();
    }
}

public function testMain() returns error? {
    http:Client c = check new ("http://localhost:19222");

    xml elem = check c->get("/db/elem");
    io:println(elem); // @output <a><b>1</b></a>

    // A resource returning xml answers with application/xml.
    http:Response raw = check c->get("/db/elem");
    io:println(raw.getContentType()); // @output application/xml

    // A declaration-prefixed document stays a sequence, so it is not an xml:Element.
    xml declared = check c->get("/db/decl");
    io:println(declared); // @output <?xml version="1.0"?><a/>
    io:println(declared is xml:Element); // @output false

    // A document with no declaration does bind to the narrower xml:Element target.
    xml single = check c->get("/db/elem");
    io:println(single is xml:Element); // @output true

    // A declared xml:Element target (not just an xml value checked with `is`) goes through
    // the conversion path rather than the fast admits() pass-through a bare xml target hits.
    xml:Element narrowed = check c->get("/db/elem");
    io:println(narrowed); // @output <a><b>1</b></a>

    xml textXml = check c->get("/db/textXml");
    io:println(textXml); // @output <a>1</a>

    xml suffixXml = check c->get("/db/suffixXml");
    io:println(suffixXml); // @output <feed/>

    // A union admitting xml selects the xml builder.
    xml|json unionTarget = check c->get("/db/elem");
    io:println(unionTarget is xml); // @output true

    // A union member narrower than xml only intersects it rather than admitting or
    // narrowing to it; the xml builder still selects, and the parsed element fits the
    // narrower member.
    xml:Element|json narrowUnion = check c->get("/db/elem");
    io:println(narrowUnion is xml:Element); // @output true

    // A multi-item sequence is still xml, but not xml:Element.
    xml seq = check c->get("/db/seqXml");
    io:println(seq); // @output <a/><b/>
    io:println(seq is xml:Element); // @output false

    // A sequence body against a declared xml:Element target fails conversion outright
    // rather than partially binding.
    xml:Element|error tooWide = c->get("/db/seqXml");
    io:println(tooWide is error); // @output true

    // An empty body binds to () for a nilable target.
    xml? optional = check c->get("/db/emptyXml");
    io:println(optional is ()); // @output true

    // A non-nilable xml target rejects an empty body rather than inventing a value.
    xml|error empty = c->get("/db/emptyXml");
    if empty is error {
        io:println(empty.message()); // @output No content
    }

    xml|error broken = c->get("/db/brokenXml");
    if broken is error {
        io:println(broken.message()); // @output Error occurred while retrieving the xml payload from the response
    }

    // application/xml has no string builder.
    string|error asString = c->get("/db/elem");
    if asString is error {
        io:println(asString.message()); // @output incompatible 'string' found for 'application/xml' mime type
    }

    // An xml request payload is sent as application/xml and round-trips.
    xml echoed = check c->post("/db/echo", xml `<p>hi</p>`);
    io:println(echoed); // @output <p>hi</p>

    string sentType = check c->post("/db/ctype", xml `<p>hi</p>`);
    io:println(sentType); // @output application/xml

    // getXmlPayload on a client-returned response uses responseMethodKeyCache.
    http:Response direct = check c->get("/db/elem");
    xml fromResponse = check direct.getXmlPayload();
    io:println(fromResponse); // @output <a><b>1</b></a>

    return;
}
