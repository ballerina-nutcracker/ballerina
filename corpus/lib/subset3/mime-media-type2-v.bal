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

public function main() {
    // Values returned by the parse functions carry their real class types.
    mime:MediaType|mime:InvalidContentTypeError parsed = mime:getMediaType("text/plain");
    io:println(parsed is mime:MediaType); // @output true

    // The full sub-type, including a structured-syntax suffix, is kept.
    mime:MediaType vnd = checkpanic mime:getMediaType("application/vnd.api+json");
    io:println(vnd.subType, " ", vnd.suffix, " ", vnd.getBaseType()); // @output vnd.api+json json application/vnd.api+json

    // Type, sub-type and parameter names are lowercased; quoted values are unquoted.
    mime:MediaType mixed = checkpanic mime:getMediaType("Text/Plain; Charset=\"UTF-8\"; a=\"x y\"");
    io:println(mixed.getBaseType(), " ", mixed.parameters["charset"], " ", mixed.parameters["a"]); // @output text/plain UTF-8 x y

    // Parameters keep a stable order across calls.
    string contentType = "multipart/form-data; type=x;q=1;start=y;charset=utf-8;boundary=abc";
    string first = (checkpanic mime:getMediaType(contentType)).toString();
    boolean stable = true;
    foreach int _ in 0 ..< 10 {
        stable = stable && (checkpanic mime:getMediaType(contentType)).toString() == first;
    }
    io:println(stable); // @output true

    // A repeated parameter is accepted; the last value wins.
    mime:MediaType|mime:InvalidContentTypeError dup = mime:getMediaType("text/plain; a=b; a=c");
    if dup is mime:MediaType {
        io:println(dup.parameters["a"]); // @output c
    }

    // A trailing semicolon and space before it are accepted.
    io:println(mime:getMediaType("text/plain ; a=1;") is mime:MediaType); // @output true

    // Malformed values are rejected.
    io:println(mime:getMediaType("invalid") is mime:InvalidContentTypeError); // @output true
    io:println(mime:getMediaType("text/") is error); // @output true
    io:println(mime:getMediaType("a/b/c") is error); // @output true
    io:println(mime:getMediaType("text/plain; a") is error); // @output true
    io:println(mime:getMediaType("text/plain; a=b c") is error); // @output true
    io:println(mime:getMediaType("") is error); // @output true
}
