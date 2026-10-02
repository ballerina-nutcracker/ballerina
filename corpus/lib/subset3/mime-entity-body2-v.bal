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
    // getJson keeps object key order and parses fractions as decimal.
    mime:Entity jsonEntity = new;
    jsonEntity.setText("{\"z\":1.5,\"y\":2,\"x\":[1e2,-3],\"w\":null}");
    map<json> parsed = <map<json>>checkpanic jsonEntity.getJson();
    io:println(parsed.keys()); // @output ["z","y","x","w"]
    io:println(parsed["z"] is decimal); // @output true
    io:println(parsed["y"] is int); // @output true
    json[] arr = <json[]>parsed["x"];
    io:println(arr[0] is decimal, " ", arr[1] is int); // @output true true

    mime:Entity badJson = new;
    badJson.setText("{\"a\" 1}");
    io:println(badJson.getJson() is mime:ParserError); // @output true

    // getXml needs a single root element; an empty body is an empty xml sequence.
    io:println(xmlResult("hello")); // @output error
    io:println(xmlResult("<a/><b/>")); // @output error
    io:println(xmlResult("<!--c-->")); // @output error
    io:println(xmlResult("  <a>t</a>  ")); // @output <a>t</a>
    io:println(xmlResult("").length()); // @output 0

    // getByteArray on an entity with no body, or with body parts, is an empty array.
    io:println((new mime:Entity()).getByteArray()); // @output []
    mime:Entity multipart = new;
    mime:Entity part = new;
    multipart.setBodyParts([part]);
    io:println(multipart.getByteArray()); // @output []

    // getHeaders returns the entity's own list, not a copy.
    mime:Entity headers = new;
    headers.addHeader("x-a", "1");
    headers.addHeader("x-a", "2");
    string[] values = checkpanic headers.getHeaders("x-a");
    values.push("3");
    io:println(headers.getHeaders("x-a")); // @output ["1","2","3"]
}

function xmlResult(string content) returns string {
    mime:Entity entity = new;
    entity.setText(content);
    xml|mime:ParserError result = entity.getXml();
    return result is xml ? result.toString() : "error";
}
