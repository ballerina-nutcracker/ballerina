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

function show(mime:Entity entity) {
    string|mime:ParserError text = entity.getText();
    io:println(text is string ? text : text.message());
    byte[]|mime:ParserError bytes = entity.getByteArray();
    if bytes is byte[] {
        io:println(string:fromBytes(bytes));
    }
}

public function main() {
    mime:Entity entity = new;
    entity.setJson({"z": 1.0, "a": "<b>&", "n": null, "t": true, "i": -3, "d": 2.50d});
    show(entity);
    // @output {"z":1.0, "a":"<b>&", "n":null, "t":true, "i":-3, "d":2.50}
    // @output {"z":1.0,"a":"<b>&","n":null,"t":true,"i":-3,"d":2.50}

    entity.setJson({"nested": {"x": [1, 2.5, -0.0], "y": {}}, "e": [], "s": "a/b"});
    show(entity);
    // @output {"nested":{"x":[1, 2.5, -0.0], "y":{}}, "e":[], "s":"a/b"}
    // @output {"nested":{"x":[1,2.5,-0.0],"y":{}},"e":[],"s":"a/b"}

    entity.setJson({"s": "q\"u\\o/t\te"});
    io:println(entity.getText()); // @output {"s":"q\"u\\o\/t\te"}

    entity.setJson("plain/x");
    show(entity);
    // @output plain/x
    // @output plain/x

    entity.setJson(());
    show(entity);
    // @output Error occurred while extracting text data from entity
    // @output
}
