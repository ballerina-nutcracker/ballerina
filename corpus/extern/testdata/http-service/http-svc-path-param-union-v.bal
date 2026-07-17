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

// A path parameter type outside the basic-type set (int/float/decimal/
// boolean/string) cannot be produced from a raw URL segment, so no resource
// should ever match it.
type Point record {|
    int x;
|};

// Exercises union-typed path parameters: the segment coercer must fall
// through to the next candidate type when an earlier one fails to parse,
// rather than rejecting the whole segment on the first parse failure.
service /api on new http:Listener(19217) {
    resource function get value/[int|float v]() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload(string `value=${v}`);
        return resp;
    }

    resource function get point/[Point r]() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload("matched");
        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19217", {});

    // "5" parses as int first.
    http:Response asInt = check c->get("/api/value/5");
    io:println(asInt.getTextPayload()); // @output value=5

    // "5.5" fails int parsing and must fall through to float.
    http:Response asFloat = check c->get("/api/value/5.5");
    io:println(asFloat.getTextPayload()); // @output value=5.5

    // No URL segment can produce a record value, so this must 404 rather
    // than incorrectly match with the raw string.
    http:Response noMatch = check c->get("/api/point/5");
    io:println(noMatch.statusCode); // @output 404
}
