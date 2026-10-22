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

// The error paths of the event-stream generators log at ERROR level, as jBallerina's do, so
// this test's stderr carries those log records with the timestamp normalized to <TIME>.
import ballerina/http;
import ballerina/io;

class FailingEvents {
    private int index = 0;

    public isolated function next() returns record {|http:SseEvent value;|}|error? {
        self.index += 1;
        if self.index == 1 {
            return {value: {data: "ok"}};
        }
        return error("db down");
    }
}

class EmptyEvent {
    public isolated function next() returns record {|http:SseEvent value;|}|error? {
        return {value: {}};
    }
}

class Bytes {
    private final (byte[]|io:Error)[] parts;
    private int index = 0;

    isolated function init((byte[]|io:Error)[] parts) {
        self.parts = parts;
    }

    public isolated function next() returns record {|byte[] value;|}|io:Error? {
        if self.index >= self.parts.length() {
            return;
        }
        byte[]|io:Error part = self.parts[self.index];
        self.index += 1;
        return part is byte[] ? {value: part} : part;
    }
}

function drain(stream<byte[], io:Error?> byteStream) returns string|error {
    string text = "";
    record {|byte[] value;|}? block = check byteStream.next();
    while block is record {|byte[] value;|} {
        text += check string:fromBytes(block.value);
        block = check byteStream.next();
    }
    return text;
}

function eventStreamResponse((byte[]|io:Error)... parts) returns http:Response {
    http:Response res = new;
    res.setByteStream(new stream<byte[], io:Error?>(new Bytes(parts)), "text/event-stream");
    return res;
}

public function main() returns error? {
    http:Response text = new;
    text.setTextPayload("not events");
    // @output error PayloadBindingClientError ("invalid payload target type. The response is not of text/event-stream content type.")
    io:println(text.getSseEventStream());

    http:Response empty = new;
    check empty.setContentType("text/event-stream");
    // @output error GenericClientError ("Error occurred while retrieving the byte stream from the response",error ParserError ("Byte stream is not available as payload"))
    io:println(empty.getSseEventStream());

    // An error from the event stream is sent as a final error event.
    http:Response failing = new;
    failing.setSseEventStream(new stream<http:SseEvent, error?>(new FailingEvents()));
    // @output ["data: ok\n\nevent: error\ndata: db down\n\n"]
    io:println([check drain(check failing.getByteStream())]);

    // So is an event with no fields.
    http:Response invalid = new;
    invalid.setSseEventStream(new stream<http:SseEvent, error?>(new EmptyEvent()));
    // @output ["event: error\ndata: Invalid value provided as event: at least one field is expected to be present in the SseEvent record.\n\n"]
    io:println([check drain(check invalid.getByteStream())]);

    // An event that is not valid UTF-8 ends the event stream with an error.
    stream<http:SseEvent, error?> badText = check eventStreamResponse([100, 97, 116, 97, 58, 255, 10, 10]).getSseEventStream();
    record {|http:SseEvent value;|}|error? badEvent = badText.next();
    io:println(badEvent is error); // @output true

    // An error from a byte stream set as the body is returned as it is.
    stream<http:SseEvent, error?> failingBytes = check eventStreamResponse("data: a\n\n".toBytes(),
        error("connection reset")).getSseEventStream();
    io:println(check failingBytes.next()); // @output {"value":{"data":"a"}}
    io:println(failingBytes.next()); // @output error("connection reset")
}
