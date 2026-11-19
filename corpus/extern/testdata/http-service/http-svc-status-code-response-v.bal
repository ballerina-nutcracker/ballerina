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

type Release record {|
    int year;
    string title;
    xml notes;
|};

type AlbumOk record {|
    *http:Ok;
    Album body;
|};

service /records on new http:Listener(19231) {
    resource function get ok() returns http:Ok {
        return {body: {title: "Blue Train", year: 1957}};
    }

    resource function get created() returns http:Created {
        return {
            body: "created",
            headers: {"x-text": "a", "x-int": 3, "x-bool": true, "x-list": ["1", "2"], "x-ints": [4, 5]}
        };
    }

    resource function get missing() returns http:NotFound {
        return {};
    }

    resource function get custom() returns http:Accepted {
        return {mediaType: " application/vnd.album+json ", body: {title: "Kind of Blue", year: 1959}};
    }

    resource function get csv() returns http:Ok {
        return {headers: {"content-type": "text/csv"}, body: "a,b"};
    }

    resource function get form() returns http:Ok {
        return {mediaType: "application/x-www-form-urlencoded", body: {name: "John Coltrane", instrument: "sax&flute"}};
    }

    resource function get bothTypes() returns http:Ok {
        return {mediaType: " text/plain ", headers: {"content-type": "text/csv"}, body: "a"};
    }

    resource function get release() returns http:Ok {
        Release release = {year: 1957, title: "Blue Train", notes: xml `<note>Blue Note</note>`};
        return {body: release};
    }

    resource function get releaseRecord() returns Release {
        return {year: 1957, title: "Blue Train", notes: xml `<note>Blue Note</note>`};
    }

    resource function get notANumber() returns http:Ok {
        return {body: {ratio: float:NaN}};
    }

    resource function get processing() returns http:Processing {
        return {body: "working"};
    }

    resource function get outOfRange() returns http:DefaultStatusCodeResponse {
        return {status: new (42), body: "x"};
    }

    resource function get badForm() returns http:Ok {
        return {mediaType: "application/x-www-form-urlencoded", body: [1, 2]};
    }

    resource function get teapot() returns http:DefaultStatusCodeResponse {
        return {status: new (418), body: "short and stout"};
    }

    resource function get bytes() returns http:Ok {
        byte[] payload = [1, 2, 3];
        return {body: payload};
    }

    resource function get markup() returns http:Ok {
        return {body: xml `<album>Giant Steps</album>`};
    }

    resource function get album() returns AlbumOk {
        return {body: {title: "Giant Steps", year: 1960}};
    }

    resource function get empty() returns http:NoContent {
        return {headers: {"x-reason": "none"}};
    }

    resource function get choice/[int id]() returns http:Ok|http:NotFound {
        if id == 1 {
            http:Ok found = {body: "found"};
            return found;
        }
        http:NotFound notFound = {body: "no album " + id.toString()};
        return notFound;
    }
}

function show(http:Response response) returns error? {
    string|error contentType = response.getHeader("content-type");
    io:println(response.statusCode, " ", contentType is string ? contentType : "-");
}

public function testMain() returns error? {
    http:Client c = check new ("http://localhost:19231/records");

    http:Response ok = check c->get("/ok");
    check show(ok); // @output 200 application/json
    io:println(check ok.getTextPayload()); // @output {"title":"Blue Train","year":1957}

    http:Response created = check c->get("/created");
    check show(created); // @output 201 text/plain
    io:println(check created.getTextPayload()); // @output created
    io:println(check created.getHeader("x-text"), " ", check created.getHeader("x-int"), " ", check created.getHeader("x-bool")); // @output a 3 true
    io:println(check created.getHeaders("x-list"), " ", check created.getHeaders("x-ints")); // @output ["1","2"] ["4","5"]

    http:Response missing = check c->get("/missing");
    check show(missing); // @output 404 -
    io:println(check missing.getBinaryPayload()); // @output []

    http:Response custom = check c->get("/custom");
    check show(custom); // @output 202 application/vnd.album+json
    io:println(check custom.getTextPayload()); // @output {"title":"Kind of Blue","year":1959}

    http:Response csv = check c->get("/csv");
    check show(csv); // @output 200 text/csv
    io:println(check csv.getTextPayload()); // @output a,b

    http:Response form = check c->get("/form");
    check show(form); // @output 200 application/x-www-form-urlencoded
    io:println(check form.getTextPayload()); // @output name=John%20Coltrane&instrument=sax%26flute

    http:Response bothTypes = check c->get("/bothTypes");
    check show(bothTypes); // @output 200 text/plain

    http:Response release = check c->get("/release");
    io:println(check release.getTextPayload()); // @output {"year":1957,"title":"Blue Train","notes":"<note>Blue Note</note>"}
    http:Response releaseRecord = check c->get("/releaseRecord");
    io:println(check releaseRecord.getTextPayload()); // @output {"year":1957,"title":"Blue Train","notes":"<note>Blue Note</note>"}

    http:Response notANumber = check c->get("/notANumber");
    // jBallerina sends invalid JSON here: https://github.com/ballerina-platform/ballerina-library/issues/9814
    io:println(notANumber.statusCode); // @output 500

    http:Response processing = check c->get("/processing");
    // jBallerina leaves the client waiting: https://github.com/ballerina-platform/ballerina-library/issues/9812
    io:println(processing.statusCode); // @output 500
    http:Response outOfRange = check c->get("/outOfRange");
    // jBallerina writes a malformed status line: https://github.com/ballerina-platform/ballerina-library/issues/9813
    io:println(outOfRange.statusCode); // @output 500

    http:Response badForm = check c->get("/badForm");
    io:println(badForm.statusCode); // @output 500

    http:Response teapot = check c->get("/teapot");
    check show(teapot); // @output 418 text/plain
    io:println(check teapot.getTextPayload()); // @output short and stout

    http:Response bytes = check c->get("/bytes");
    check show(bytes); // @output 200 application/octet-stream
    io:println(check bytes.getBinaryPayload()); // @output [1,2,3]

    http:Response markup = check c->get("/markup");
    check show(markup); // @output 200 application/xml
    io:println(check markup.getTextPayload()); // @output <album>Giant Steps</album>

    http:Response album = check c->get("/album");
    check show(album); // @output 200 application/json
    io:println(check album.getTextPayload()); // @output {"title":"Giant Steps","year":1960}

    http:Response empty = check c->get("/empty");
    check show(empty); // @output 204 -
    io:println(check empty.getHeader("x-reason")); // @output none

    http:Response found = check c->get("/choice/1");
    check show(found); // @output 200 text/plain
    http:Response notFound = check c->get("/choice/2");
    check show(notFound); // @output 404 text/plain
    io:println(check notFound.getTextPayload()); // @output no album 2
}
