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

function show(string contentType) {
    mime:MediaType|mime:InvalidContentTypeError result = mime:getMediaType(contentType);
    io:println(result is error ? result.message() : result.toString());
}

public function main() {
    show("text"); // @output Unable to find a sub type.
    show(""); // @output Unable to find a sub type.
    show("text;a=b/c"); // @output Unable to find a sub type.
    show("/plain"); // @output Primary type is invalid.
    show("te@xt/plain"); // @output Primary type is invalid.
    show("text/"); // @output Sub type is invalid.
    show("text/pl ain"); // @output Sub type is invalid.
    show("text/plain; charset"); // @output Couldn't find the '=' that separates a parameter name from its value.
    show("text/plain; a=\"x"); // @output Encountered unterminated quoted parameter value.
    show("text/plain; a="); // @output Couldn't find a value for parameter named a
    show("text/plain; a=b c"); // @output More characters encountered in input than expected.
    show("text/plain; a=@"); // @output Unexpected character encountered at index 4
    show("te@xt/plain; a"); // @output Couldn't find the '=' that separates a parameter name from its value.
    show("text/plain;"); // @output text/plain
    show("text/plain; =x"); // @output text/plain; =x
}
