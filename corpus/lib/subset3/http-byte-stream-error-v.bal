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
import ballerina/mime;

class OneChunk {
    private boolean done = false;

    public isolated function next() returns record {|byte[] value;|}|io:Error? {
        if self.done {
            return;
        }
        self.done = true;
        return {value: "abc".toBytes()};
    }
}

public function main() {
    http:Response res = new;
    res.setTextPayload("text");
    // @output error GenericClientError ("Error occurred while retrieving the byte stream from the response",error GenericMimeError ("The array size of a byte stream must be positive, but was 0"))
    io:println(res.getByteStream(0));
    // @output error GenericClientError ("Error occurred while retrieving the byte stream from the response",error GenericMimeError ("The array size of a byte stream must be positive, but was -4"))
    io:println(res.getByteStream(-4));

    // A payload set by a payload setter is not a byte stream.
    // @output error GenericClientError ("Error occurred while retrieving the byte stream from the response",error ParserError ("Byte stream is not available but payload can be obtain either as xml, json, string or byte[] type"))
    io:println(res.getByteStream());

    http:Request req = new;
    // @output error GenericClientError ("Error occurred while retrieving the byte stream from the request",error ParserError ("Byte stream is not available as payload"))
    io:println(req.getByteStream());

    mime:Entity part = new;
    part.setText("part");
    http:Request multipart = new;
    multipart.setBodyParts([part]);
    // @output error GenericClientError ("Error occurred while retrieving the byte stream from the request",error ParserError ("Byte stream is not available since payload contains a set of body parts"))
    io:println(multipart.getByteStream());

    // As in jBallerina, the text, json and xml payload getters see a body set with
    // setByteStream as empty content.
    http:Request streamed = new;
    streamed.setByteStream(new stream<byte[], io:Error?>(new OneChunk()));
    // @output error NoContentError ("No content",error ParserError ("Error occurred while extracting text data from entity",error NoContentError ("Empty content")))
    io:println(streamed.getTextPayload());
    http:Response streamedJson = new;
    streamedJson.setByteStream(new stream<byte[], io:Error?>(new OneChunk()), "application/json");
    // @output error NoContentError ("No content",error ParserError ("Error occurred while extracting json data from entity",error NoContentError ("Empty content")))
    io:println(streamedJson.getJsonPayload());
    // @output error NoContentError ("No content",error ParserError ("Error occurred while extracting text data from entity",error NoContentError ("Empty content")))
    io:println(streamedJson.getXmlPayload());
}
