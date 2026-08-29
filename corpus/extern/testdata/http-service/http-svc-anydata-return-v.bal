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
    string title;
    int year;
|};

// A non-Response object is not an http:Response — the return type is technically
// compilable (only a function-typed or client-object return is rejected), but writeResult
// must still reject it rather than treating it as a serialisable anydata value.
class Widget {
    int id = 1;
}

service /svc on new http:Listener(19223) {
    resource function get xmlBody() returns xml {
        return xml `<a><b>1</b></a>`;
    }

    resource function get text() returns string {
        return "hello";
    }

    resource function get bytes() returns byte[] {
        return [104, 105];
    }

    resource function get num() returns int {
        return 42;
    }

    resource function get flag() returns boolean {
        return true;
    }

    resource function get rec() returns Album {
        return {title: "Kind of Blue", year: 1959};
    }

    // A json[] is not a byte[], so it must take the json branch rather than the binary one.
    resource function get jsonArr() returns json[] {
        return [1, 2, 3];
    }

    // A non-nil anydata return from a `post` resource is 201 Created.
    resource function post made() returns string {
        return "created";
    }

    // The 201 rule keys on the declared accessor, so a `default` resource reached by a
    // POST stays 200.
    resource function default anyVerb() returns string {
        return "any verb";
    }

    // A () return stays 202 Accepted regardless of accessor.
    resource function post nothing() returns error? {
        return;
    }

    // An error return is still the 500 envelope.
    resource function get boom() returns xml|error {
        return error("kaboom");
    }

    // A value that fails to serialize as JSON (float:NaN has no JSON representation)
    // reports the failure rather than sending a malformed body.
    resource function get notFinite() returns float {
        return float:NaN;
    }

    // A non-Response object return is rejected, not silently treated as anydata.
    resource function get widget() returns Widget {
        return new Widget();
    }
}

public function testMain() returns error? {
    http:Client c = check new ("http://localhost:19223");

    http:Response xmlRes = check c->get("/svc/xmlBody");
    io:println(xmlRes.statusCode); // @output 200
    io:println(xmlRes.getContentType()); // @output application/xml
    io:println(check xmlRes.getTextPayload()); // @output <a><b>1</b></a>

    http:Response textRes = check c->get("/svc/text");
    io:println(textRes.getContentType()); // @output text/plain
    io:println(check textRes.getTextPayload()); // @output hello

    http:Response bytesRes = check c->get("/svc/bytes");
    io:println(bytesRes.getContentType()); // @output application/octet-stream
    io:println(check bytesRes.getTextPayload()); // @output hi

    http:Response numRes = check c->get("/svc/num");
    io:println(numRes.getContentType()); // @output application/json
    io:println(check numRes.getTextPayload()); // @output 42

    http:Response flagRes = check c->get("/svc/flag");
    io:println(flagRes.getContentType()); // @output application/json
    io:println(check flagRes.getTextPayload()); // @output true

    http:Response recRes = check c->get("/svc/rec");
    io:println(recRes.getContentType()); // @output application/json
    io:println(check recRes.getTextPayload()); // @output {"title":"Kind of Blue","year":1959}

    http:Response arrRes = check c->get("/svc/jsonArr");
    io:println(arrRes.getContentType()); // @output application/json
    io:println(check arrRes.getTextPayload()); // @output [1,2,3]

    // The client binds these returns directly to the expected type.
    xml boundXml = check c->get("/svc/xmlBody");
    io:println(boundXml); // @output <a><b>1</b></a>
    Album boundRec = check c->get("/svc/rec");
    io:println(boundRec.title); // @output Kind of Blue

    http:Response createdRes = check c->post("/svc/made", ());
    io:println(createdRes.statusCode); // @output 201

    http:Response defaultRes = check c->post("/svc/anyVerb", ());
    io:println(defaultRes.statusCode); // @output 200

    http:Response nilRes = check c->post("/svc/nothing", ());
    io:println(nilRes.statusCode); // @output 202

    http:Response errRes = check c->get("/svc/boom");
    io:println(errRes.statusCode); // @output 500

    http:Response nanRes = check c->get("/svc/notFinite");
    io:println(nanRes.statusCode); // @output 500

    http:Response widgetRes = check c->get("/svc/widget");
    io:println(widgetRes.statusCode); // @output 500

    return;
}
