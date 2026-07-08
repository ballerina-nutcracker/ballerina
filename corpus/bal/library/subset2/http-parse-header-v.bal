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
    var values = check http:parseHeader("text/plain;q=0.9, application/json");
    io:println(values.length()); // @output 2

    // Quoted param value: boundary should be stripped of quotes.
    var quoted = check http:parseHeader("multipart/form-data; boundary=\"----boundary\"");
    io:println(quoted.length()); // @output 1
    io:println(quoted[0].value); // @output multipart/form-data
    io:println(quoted[0].params["boundary"]); // @output ----boundary

    // Param without value: boundary key present with empty string.
    var noVal = check http:parseHeader("multipart/form-data; boundary");
    io:println(noVal.length()); // @output 1
    io:println(noVal[0].params["boundary"]); // @output

    // Single value with no params.
    var single = check http:parseHeader("text/html");
    io:println(single.length()); // @output 1
    return;
}
