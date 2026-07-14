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
    http:Client c = check new http:Client("http://testserver", {});

    // POST a JSON body spanning every value kind (string, int, float, decimal,
    // bool, null, nested list, nested map) -> exercises balToGoJSON. The server
    // echoes the serialized body back in the map's insertion order.
    json reqBody = {
        "s": "hi",
        "i": 7,
        "f": 2.5,
        "d": 3.5d,
        "b": true,
        "n": (),
        "arr": [1, 2, 3],
        "obj": {"k": "v"}
    };
    http:Response r = check c->post("/echo", reqBody);
    io:println(r.getTextPayload()); // @output {"s":"hi","i":7,"f":2.5,"d":3.5,"b":true,"n":null,"arr":[1,2,3],"obj":{"k":"v"}}

    // GET a JSON array response with mixed element types -> exercises goToBalValue
    // for integer, float, string, bool, null, nested array, and nested object.
    http:Response r2 = check c->get("/json-array");
    json payload = check r2.getJsonPayload();
    io:println(payload); // @output [1,2.5,"three",true,null,[10,20],{"k":"v"}]
    return;
}
