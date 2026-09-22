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

type Person record {|
    string name;
    int age;
|};

// A tuple is a subtype of byte[], so the blob builder is selected and must reject a body
// whose length does not fit.
type Pair [byte, byte];

// A closed all-string record is a subtype of map<string>, so the form builder is selected
// and must reject a map that does not fit the record.
type Form record {|
    string a;
    string b;
|};

enum Colour {
    RED = "red",
    GREEN = "green"
}

service /db on new http:Listener(19221) {
    resource function get missing() returns http:Response {
        http:Response r = new;
        r.statusCode = 404;
        r.setJsonPayload({"error": "gone"});
        return r;
    }

    resource function get boom() returns http:Response {
        http:Response r = new;
        r.statusCode = 500;
        r.setTextPayload("kaboom");
        return r;
    }

    resource function get moved() returns http:Response {
        http:Response r = new;
        r.statusCode = 302;
        r.setTextPayload("moved body");
        return r;
    }

    resource function get count() returns http:Response {
        http:Response r = new;
        r.setJsonPayload(7);
        return r;
    }

    resource function get brokenJson() returns http:Response {
        http:Response r = new;
        r.setTextPayload("{\"name\": \"Alice\", \"age\": nope}", "application/json");
        return r;
    }

    resource function get trailingJson() returns http:Response {
        http:Response r = new;
        r.setTextPayload("{\"name\": \"Alice\", \"age\": 30}trailing", "application/json");
        return r;
    }

    resource function get text() returns http:Response {
        http:Response r = new;
        r.setTextPayload("plain text body");
        return r;
    }

    resource function get xmlBody() returns http:Response {
        http:Response r = new;
        r.setTextPayload("<a>1</a>", "application/xml");
        return r;
    }

    // A 4xx whose error detail body is application/xml is still parsed as xml.
    resource function get missingXml() returns http:Response {
        http:Response r = new;
        r.statusCode = 404;
        r.setTextPayload("<error>gone</error>", "application/xml");
        return r;
    }

    resource function get blob() returns http:Response {
        http:Response r = new;
        r.setBinaryPayload([1, 2, 3]);
        return r;
    }

    resource function get form() returns http:Response {
        http:Response r = new;
        r.setTextPayload("a=1&b=two", "application/x-www-form-urlencoded");
        return r;
    }

    resource function get badForm() returns http:Response {
        http:Response r = new;
        r.setTextPayload("a=%zz", "application/x-www-form-urlencoded");
        return r;
    }

    resource function get goneBlob() returns http:Response {
        http:Response r = new;
        r.statusCode = 410;
        r.setBinaryPayload([9]);
        return r;
    }

    // 499 is nginx's client-closed-request code and has no registered reason phrase.
    resource function get nginx499() returns http:Response {
        http:Response r = new;
        r.statusCode = 499;
        r.setTextPayload("closed");
        return r;
    }

    // A status error carrying a JSON content type but no body at all.
    resource function get emptyJson401() returns http:Response {
        http:Response r = new;
        r.statusCode = 401;
        r.setTextPayload("", "application/json");
        return r;
    }

    // A status error whose declared JSON body does not parse.
    resource function get missingBrokenJson() returns http:Response {
        http:Response r = new;
        r.statusCode = 404;
        r.setTextPayload("{\"error\": nope}", "application/json");
        return r;
    }

    resource function get empty() returns http:Response {
        http:Response r = new;
        r.setTextPayload("");
        return r;
    }

    resource function get emptyBlob() returns http:Response {
        http:Response r = new;
        byte[] none = [];
        r.setBinaryPayload(none);
        return r;
    }

    resource function get emptyForm() returns http:Response {
        http:Response r = new;
        r.setTextPayload("", "application/x-www-form-urlencoded");
        return r;
    }

    // Non-empty on the wire but decodes to an empty map: must not be treated as absent.
    resource function get blankForm() returns http:Response {
        http:Response r = new;
        r.setTextPayload("&", "application/x-www-form-urlencoded");
        return r;
    }

    resource function get emptyJson() returns http:Response {
        http:Response r = new;
        r.setTextPayload("", "application/json");
        return r;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19221", {});

    // A 4xx response becomes an error whose message is the reason phrase.
    Person|error notFound = c->get("/db/missing");
    if notFound is error {
        io:println("4xx: ", notFound.message()); // @output 4xx: Not Found
    }

    // A 5xx response likewise.
    Person|error serverErr = c->get("/db/boom");
    if serverErr is error {
        io:println("5xx: ", serverErr.message()); // @output 5xx: Internal Server Error
    }

    // A 4xx whose error detail body is application/xml is still read via the xml builder,
    // not the text fallback.
    Person|error xmlErrBody = c->get("/db/missingXml");
    if xmlErrBody is error {
        io:println("4xx-xml: ", xmlErrBody.message()); // @output 4xx-xml: Not Found
    }

    // An http:Response target bypasses the status-code mapping entirely.
    http:Response raw = check c->get("/db/missing");
    io:println(raw.statusCode); // @output 404

    // A 3xx response is not an error, so binding still runs.
    string redirected = check c->get("/db/moved");
    io:println(redirected); // @output moved body

    // A payload that does not fit the target type fails binding.
    Person|error mismatch = c->get("/db/count");
    if mismatch is error {
        io:println(mismatch.message()); // @output Payload binding failed: '7' value cannot be converted to '{| age: int, name: string, never... |}'
    }

    // Malformed JSON fails to parse.
    Person|error malformed = c->get("/db/brokenJson");
    if malformed is error {
        io:println(malformed.message()); // @output failed to parse JSON payload: invalid character 'o' in literal null (expecting 'u')
    }

    // Trailing data after an otherwise well-formed JSON value is rejected rather than
    // silently ignored.
    Person|error trailing = c->get("/db/trailingJson");
    if trailing is error {
        io:println(trailing.message()); // @output failed to parse JSON payload: invalid character 'a' in literal true (expecting 'u')
    }

    // getJsonPayload shares the same decoder, so it rejects trailing data too.
    http:Response trailingResp = check c->get("/db/trailingJson");
    json|error trailingJson = trailingResp.getJsonPayload();
    if trailingJson is error {
        io:println(trailingJson.message()); // @output failed to parse JSON payload: invalid character 'a' in literal true (expecting 'u')
    }

    // A text/plain response cannot bind to a record.
    Person|error wrongMime = c->get("/db/text");
    if wrongMime is error {
        io:println(wrongMime.message()); // @output incompatible '{| age: int, name: string, never... |}' found for 'text/plain' mime type
    }

    // An application/xml response has no string builder, so a string target is rejected.
    string|error asString = c->get("/db/xmlBody");
    if asString is error {
        io:println(asString.message()); // @output incompatible 'string' found for 'application/xml' mime type
    }

    // A body that is not a member of an enum target fails conversion.
    Colour|error notAColour = c->get("/db/text");
    if notAColour is error {
        io:println(notAColour.message()); // @output Payload binding failed: '"plain text body"' value cannot be converted to '"green"|"red"'
    }

    // A three-byte body does not fit a two-element tuple target.
    Pair|error tooLong = c->get("/db/blob");
    if tooLong is error {
        io:println(tooLong.message()); // @output Payload binding failed: '[int:Unsigned8...]' value cannot be converted to '[int:Unsigned8, int:Unsigned8, never...]'
    }

    // The nilable form fails the same way; only an absent body binds to ().
    Colour?|error notAColourOpt = c->get("/db/text");
    if notAColourOpt is error {
        io:println(notAColourOpt.message()); // @output Payload binding failed: '"plain text body"' value cannot be converted to 'nil|"green"|"red"'
    }

    Pair?|error tooLongOpt = c->get("/db/blob");
    if tooLongOpt is error {
        io:println(tooLongOpt.message()); // @output Payload binding failed: '[int:Unsigned8...]' value cannot be converted to 'nil|[int:Unsigned8, int:Unsigned8, never...]'
    }

    // An octet-stream response cannot bind to a string.
    string|error wrongBlob = c->get("/db/blob");
    if wrongBlob is error {
        io:println(wrongBlob.message()); // @output incompatible 'string' found for 'application/octet-stream' mime type
    }

    // A form-urlencoded response cannot bind to an int.
    int|error wrongForm = c->get("/db/form");
    if wrongForm is error {
        io:println(wrongForm.message()); // @output incompatible 'int' found for 'application/x-www-form-urlencoded' mime type
    }

    // A malformed form-urlencoded body fails to parse.
    map<string>|error badForm = c->get("/db/badForm");
    if badForm is error {
        io:println(badForm.message()); // @output Payload binding failed: invalid URL escape "%zz"
    }

    // A 4xx body is extracted according to its own media type.
    byte[]|error goneBlob = c->get("/db/goneBlob");
    if goneBlob is error {
        io:println(goneBlob.message()); // @output Gone
    }

    // A code with no registered phrase names the code, so the message is never empty.
    string|error unregistered = c->get("/db/nginx499");
    if unregistered is error {
        io:println(unregistered.message()); // @output status code 499
    }

    // A status error with a JSON content type but no body keeps its reason phrase.
    Person|error emptyJsonError = c->get("/db/emptyJson401");
    if emptyJsonError is error {
        io:println(emptyJsonError.message()); // @output Unauthorized
    }

    // A status error whose body is present but unparsable reports the extraction failure.
    Person|error brokenJsonError = c->get("/db/missingBrokenJson");
    if brokenJsonError is error {
        io:println(brokenJsonError.message()); // @output http:ApplicationResponseError creation failed: 404 response payload extraction failed
    }

    // An empty body binds to () for a nilable target.
    string? emptyText = check c->get("/db/empty");
    io:println(emptyText is ()); // @output true

    // An empty octet-stream body binds to () for a nilable byte[] target.
    byte[]? emptyBlob = check c->get("/db/emptyBlob");
    io:println(emptyBlob is ()); // @output true

    // An empty form body binds to () for a nilable map<string> target.
    map<string>? emptyForm = check c->get("/db/emptyForm");
    io:println(emptyForm is ()); // @output true

    // A body of "&" is non-empty on the wire even though it decodes to an empty map, so it
    // must still bind to that empty map rather than ().
    map<string>? blankForm = check c->get("/db/blankForm");
    io:println(blankForm is ()); // @output false

    // The absent-payload rule is decided before the Content-Type picks a builder, so it holds
    // for a nilable target the selected builder would otherwise reject outright.
    int? emptyInt = check c->get("/db/empty");
    io:println(emptyInt is ()); // @output true

    map<json>? emptyJsonMap = check c->get("/db/empty");
    io:println(emptyJsonMap is ()); // @output true

    // A non-nilable target the builder rejects is still an incompatible-target error.
    int|error nonNilInt = c->get("/db/empty");
    if nonNilInt is error {
        io:println(nonNilInt.message()); // @output incompatible 'int' found for 'text/plain' mime type
    }

    // An empty body with a non-nilable json target fails to parse.
    Person|error emptyRecord = c->get("/db/emptyJson");
    if emptyRecord is error {
        io:println(emptyRecord.message()); // @output failed to parse JSON payload: EOF
    }

    // An empty body binds to () for a nilable record target.
    Person? emptyOptional = check c->get("/db/emptyJson");
    io:println(emptyOptional is ()); // @output true

    // Only a nilable target turns an absent body into (). A narrower-than-builder target that
    // is not nilable is handed the builder's empty value and must reject it, rather than
    // admitting "" or [] or {} as a member of the narrow type. One case per builder.
    Colour|error emptyColour = c->get("/db/empty");
    if emptyColour is error {
        io:println(emptyColour.message()); // @output Payload binding failed: '""' value cannot be converted to '"green"|"red"'
    }

    Pair|error emptyPair = c->get("/db/emptyBlob");
    if emptyPair is error {
        io:println(emptyPair.message()); // @output Payload binding failed: '[int:Unsigned8...]' value cannot be converted to '[int:Unsigned8, int:Unsigned8, never...]'
    }

    Form|error emptyFormRecord = c->get("/db/emptyForm");
    if emptyFormRecord is error {
        io:println(emptyFormRecord.message()); // @output Payload binding failed: '{| string... |}' value cannot be converted to '{| a: string, b: string, never... |}': field 'a' not present in value
    }

    return;
}
