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

import ballerina/io;
import ballerina/mime;

public function main() returns error? {
    mime:MediaType result = check mime:getMediaType("application/json; charset=UTF-8");
    io:println(result.primaryType);
    io:println(result.subType);
    io:println(result.suffix);
    io:println(result.getBaseType());
    io:println(result.parameters["charset"]);

    mime:MediaType result2 = check mime:getMediaType("application/svg+xml");
    io:println(result2.primaryType);
    io:println(result2.subType);
    io:println(result2.suffix);
    io:println(result2.getBaseType());

    mime:MediaType|error result3 = mime:getMediaType("invalid!!");
    if result3 is mime:MediaType {
        io:println("got media type: ", result3);
    }
    io:println("invalid content type");
}

// @output application
// @output json
// @output
// @output application/json
// @output UTF-8
// @output application
// @output svg
// @output xml
// @output application/svg
// @output invalid content type
