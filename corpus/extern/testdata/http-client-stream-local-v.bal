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

class Chunks {
    private final string[] parts;
    private int index = 0;

    isolated function init(string[] parts) {
        self.parts = parts;
    }

    public isolated function next() returns record {|byte[] value;|}|io:Error? {
        if self.index >= self.parts.length() {
            return;
        }
        string part = self.parts[self.index];
        self.index += 1;
        return {value: part.toBytes()};
    }
}

class Failing {
    private boolean sent = false;

    public isolated function next() returns record {|byte[] value;|}|io:Error? {
        if self.sent {
            return error("boom");
        }
        self.sent = true;
        return {value: "partial".toBytes()};
    }
}

function chunks(string... parts) returns stream<byte[], io:Error?> {
    return new (new Chunks(parts));
}

function blockSizes(stream<byte[], io:Error?> byteStream) returns int[]|error {
    int[] sizes = [];
    record {|byte[] value;|}? block = check byteStream.next();
    while block is record {|byte[] value;|} {
        sizes.push(block.value.length());
        block = check byteStream.next();
    }
    return sizes;
}

function blocks(stream<byte[], io:Error?> byteStream) returns string|error {
    string text = "";
    record {|byte[] value;|}? block = check byteStream.next();
    while block is record {|byte[] value;|} {
        text += check string:fromBytes(block.value) + "|";
        block = check byteStream.next();
    }
    return text;
}

public function main() returns error? {
    http:Client c = check new ("http://testserver");

    // ---- byte-stream request bodies are sent chunked ----
    string echoed = check c->post("/echo", chunks("ab", "cd", "ef"));
    io:println(echoed); // @output abcdef|application/octet-stream|chunked=true
    string typed = check c->post("/echo", chunks("x"), mediaType = "text/csv");
    io:println(typed); // @output x|text/csv|chunked=true

    http:Request putReq = new;
    putReq.setByteStream(chunks("put ", "body"), "text/plain");
    string put = check c->execute("PUT", "/echo", putReq);
    io:println(put); // @output put body|text/plain|chunked=true

    http:Request forwardReq = new;
    forwardReq.method = "POST";
    forwardReq.setByteStream(chunks("forwarded"));
    string forwarded = check c->forward("/echo", forwardReq);
    io:println(forwarded); // @output forwarded|application/octet-stream|chunked=true

    string|error failed = c->post("/echo", new stream<byte[], io:Error?>(new Failing()));
    io:println(failed is error ? failed.message() : failed); // @output Error occurred while streaming content: boom

    // ---- reading response bodies as byte streams ----
    http:Response download = check c->get("/download");
    io:println(blockSizes(check download.getByteStream())); // @output [8192,8192,3616]

    http:Response small = check c->get("/small");
    io:println(blocks(check small.getByteStream(2))); // @output sm|al|l|

    // A second byte stream over the same body carries on where the first left off.
    http:Response twice = check c->get("/small");
    stream<byte[], io:Error?> first = check twice.getByteStream(2);
    record {|byte[] value;|}? firstBlock = check first.next();
    io:println(firstBlock is () ? "" : check string:fromBytes(firstBlock.value)); // @output sm
    io:println(blocks(check twice.getByteStream(2))); // @output al|l|

    http:Response empty = check c->get("/empty");
    // @output error GenericClientError ("Error occurred while retrieving the byte stream from the response",error ParserError ("Byte stream is not available as payload"))
    io:println(empty.getByteStream());

    http:Response read = check c->get("/small");
    _ = check read.getTextPayload();
    // @output error GenericClientError ("Error occurred while retrieving the byte stream from the response",error ParserError ("Byte stream is not available but payload can be obtain either as xml, json, string or byte[] type"))
    io:println(read.getByteStream());

    // ---- binding a text/event-stream response to an event stream ----
    stream<http:SseEvent, error?> events = check c->get("/events");
    // @output {"data":"one"}
    // @output {"data":"two\nmore","id":"2","event":"tick"}
    // @output {"data":"three","comment":"note","retry":100}
    record {|http:SseEvent value;|}? event = check events.next();
    while event is record {|http:SseEvent value;|} {
        io:println(event.value);
        event = check events.next();
    }

    stream<http:SseEvent, error?>|error notEvents = c->get("/small");
    // @output error PayloadBindingClientError ("invalid payload target type. The response is not of text/event-stream content type.")
    io:println(notEvents);

    stream<http:SseEvent, error?>|string|error mixed = c->get("/events");
    // @output error PayloadBindingClientError ("payload binding failed: Target return type must not be a union of stream<http:SseEvent, error?> and anydata")
    io:println(mixed);

    stream<http:SseEvent, error?>|error missing = c->get("/missing");
    io:println(missing is error ? missing.message() : "stream"); // @output Not Found
}
