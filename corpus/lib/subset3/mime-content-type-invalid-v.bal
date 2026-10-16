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

function show(error? result) {
    io:println(result is mime:InvalidContentTypeError ? result.message() : "ok");
}

public function main() {
    string bad = "multipart/form-data; boundary=a=b";
    mime:Entity entity = new;
    show(trap entity.setText("x", bad)); // @output More characters encountered in input than expected.
    show(trap entity.setJson({}, bad)); // @output More characters encountered in input than expected.
    show(trap entity.setXml(xml `<a/>`, bad)); // @output More characters encountered in input than expected.
    show(trap entity.setByteArray([1], bad)); // @output More characters encountered in input than expected.
    show(trap entity.setBodyParts([], bad)); // @output More characters encountered in input than expected.
    show(trap entity.setText("x", "garbage")); // @output Unable to find a sub type.
    show(trap entity.setText("x", "")); // @output Unable to find a sub type.
    show(trap entity.setText("x", "text/plain; charset=utf-8")); // @output ok
    io:println(entity.getContentType()); // @output text/plain; charset=utf-8
}
