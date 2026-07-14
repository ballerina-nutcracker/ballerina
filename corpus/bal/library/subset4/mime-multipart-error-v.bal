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

import ballerina/mime;
import ballerina/io;

public function main() returns error? {
    // non-multipart entity
    mime:Entity plain = new;
    plain.setText("just text");
    mime:Entity[]|mime:ParserError plainResult = plain.getBodyParts();
    io:println(plainResult is mime:ParserError); // @output true
    if plainResult is mime:ParserError {
        io:println(plainResult.message()); // @output Entity body is not a type of composite media type. Received content-type : text/plain
    }

    // multipart content-type with no boundary parameter
    mime:Entity noBoundary = new;
    noBoundary.setByteArray("--x\r\n--x--".toBytes(), "multipart/form-data");
    mime:Entity[]|mime:ParserError noBoundaryResult = noBoundary.getBodyParts();
    io:println(noBoundaryResult is mime:ParserError); // @output true
    if noBoundaryResult is mime:ParserError {
        io:println(noBoundaryResult.message()); // @output Error occurred while extracting body parts from entity: no boundary parameter found in Content-Type
    }

    // entity with no content-type header at all
    mime:Entity noHeader = new;
    mime:Entity[]|mime:ParserError noHeaderResult = noHeader.getBodyParts();
    io:println(noHeaderResult is mime:ParserError); // @output true
    return;
}
