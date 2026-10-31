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
import ballerina/mime;
import ballerina/io;

public function main() returns error? {
    http:Request plainReq = new;
    plainReq.setTextPayload("just text");
    mime:Entity[]|error plainReqResult = plainReq.getBodyParts();
    io:println(plainReqResult is error); // @output true
    if plainReqResult is error {
        io:println(plainReqResult.message()); // @output Error occurred while retrieving body parts from the request
    }

    http:Response plainRes = new;
    plainRes.setTextPayload("just text");
    mime:Entity[]|error plainResResult = plainRes.getBodyParts();
    io:println(plainResResult is error); // @output true
    if plainResResult is error {
        io:println(plainResResult.message()); // @output Error occurred while retrieving body parts from the response: Entity body is not a type of composite media type. Received content-type : text/plain
    }

    http:Request noPartReq = new;
    noPartReq.setTextPayload("garbage", "multipart/form-data; boundary=zzz");
    mime:Entity[]|error noPartReqResult = noPartReq.getBodyParts();
    if noPartReqResult is error {
        io:println(noPartReqResult.message()); // @output Error occurred while retrieving body parts from the request
    }

    http:Response noPartRes = new;
    noPartRes.setBinaryPayload("garbage".toBytes(), "multipart/form-data; boundary=zzz");
    mime:Entity[]|error noPartResResult = noPartRes.getBodyParts();
    if noPartResResult is error {
        io:println(noPartResResult.message()); // @output Error occurred while retrieving body parts from the response: Error occurred while extracting body parts from entity: Missing start boundary
    }
    return;
}
