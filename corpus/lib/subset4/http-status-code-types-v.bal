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
|};

type AlbumNotFound record {|
    *http:NotFound;
    Album body;
|};

function describe(http:StatusCodeResponse response) returns string {
    if response is http:Ok {
        return "Ok";
    }
    if response is http:NotFound {
        return "NotFound";
    }
    if response is http:DefaultStatusCodeResponse {
        return "Default";
    }
    return "other";
}

public function main() {
    http:Ok ok = {};
    io:println(ok.status.code, " ", ok.status is http:StatusOK, " ", ok.status is http:Status); // @output 200 true true

    http:Created created = {body: "made", mediaType: "text/plain", headers: {"x-id": 7}};
    io:println(created.status.code, " ", created?.body, " ", created?.mediaType, " ", created?.headers); // @output 201 made text/plain {"x-id":7}

    http:NoContent noContent = {headers: {"x-a": "b"}};
    io:println(noContent.status.code); // @output 204

    AlbumNotFound notFound = {body: {title: "Lost"}};
    io:println(notFound.status.code, " ", notFound.body.title, " ", notFound is http:NotFound); // @output 404 Lost true

    http:DefaultStatusCodeResponse teapot = {status: new (418)};
    io:println(teapot.status.code, " ", teapot.status is http:DefaultStatus); // @output 418 true

    http:NetworkAuthorizationRequired deprecated = {};
    io:println(deprecated.status.code); // @output 511

    io:println(describe(ok), " ", describe(notFound), " ", describe(teapot), " ", describe(created)); // @output Ok NotFound Default other
    io:println(http:STATUS_OK, " ", http:STATUS_NOT_FOUND, " ", http:STATUS_NETWORK_AUTHENTICATION_REQUIRED); // @output 200 404 511

    http:StatusInternalServerError status = new;
    io:println(status.code); // @output 500
}
