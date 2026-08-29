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
|};

enum Colour {
    RED = "red",
    GREEN = "green"
}

// The server truncates every body. The transport error text carries the ephemeral port, so
// only the fact that an error surfaced is asserted.
public function main() returns error? {
    http:Client c = check new http:Client("http://testserver", {});

    Person|error viaJson = c->get("/trunc-json");
    io:println(viaJson is error); // @output true

    string|error viaText = c->get("/trunc-text");
    io:println(viaText is error); // @output true

    xml|error viaXml = c->get("/trunc-xml");
    io:println(viaXml is error); // @output true

    // A raw http:Response target never reads the body, so the failure only surfaces once
    // getXmlPayload() is called directly on it.
    http:Response rawXml = check c->get("/trunc-xml");
    xml|error direct = rawXml.getXmlPayload();
    io:println(direct is error); // @output true

    // A narrow target adds a conversion step the read failure has to survive.
    Colour|error viaNarrowText = c->get("/trunc-text");
    io:println(viaNarrowText is error); // @output true

    byte[]|error viaBlob = c->get("/trunc-blob");
    io:println(viaBlob is error); // @output true

    map<string>|error viaForm = c->get("/trunc-form");
    io:println(viaForm is error); // @output true

    // A () target discards the body without inspecting the Content-Type, but must still
    // read it, so a read failure surfaces as an error rather than a silent ().
    ()|error viaNil = c->get("/trunc-nil");
    io:println(viaNil is error); // @output true

    // A 4xx whose detail body cannot be read reports the extraction failure.
    Person|error viaStatus = c->get("/trunc-404");
    if viaStatus is error {
        io:println(viaStatus.message()); // @output http:ApplicationResponseError creation failed: 404 response payload extraction failed
    }

    return;
}
