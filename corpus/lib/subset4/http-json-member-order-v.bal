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
    string text = "{\"name\":\"ballerina\",\"version\":1,\"kind\":\"lang\",\"m\":{\"z\":1,\"y\":2,\"x\":3,\"w\":4},\"l\":[{\"q\":1,\"p\":2}],\"e\":5,\"d\":6,\"c\":7,\"b\":8,\"a\":9}";
    http:Request req = new;
    req.setTextPayload(text, "application/json");
    json payload = check req.getJsonPayload();
    http:Response resp = new;
    resp.setJsonPayload(payload);
    json again = check resp.getJsonPayload();
    io:println((<map<json>>payload).keys()); // @output ["name","version","kind","m","l","e","d","c","b","a"]
    io:println((<map<json>>again).keys()); // @output ["name","version","kind","m","l","e","d","c","b","a"]
    io:println(again == payload); // @output true
    io:println(check again.m); // @output {"z":1,"y":2,"x":3,"w":4}
}
