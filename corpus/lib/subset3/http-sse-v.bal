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

class Events {
    private final http:SseEvent[] events;
    private int index = 0;

    isolated function init(http:SseEvent[] events) {
        self.events = events;
    }

    public isolated function next() returns record {|http:SseEvent value;|}|error? {
        if self.index >= self.events.length() {
            return;
        }
        http:SseEvent event = self.events[self.index];
        self.index += 1;
        return {value: event};
    }
}

class Chunks {
    private final byte[][] parts;
    private int index = 0;

    isolated function init(byte[][] parts) {
        self.parts = parts;
    }

    public isolated function next() returns record {|byte[] value;|}|io:Error? {
        if self.index >= self.parts.length() {
            return;
        }
        byte[] part = self.parts[self.index];
        self.index += 1;
        return {value: part};
    }
}

type EventStream stream<http:SseEvent, error?>;

function bytesBetween(byte[] bytes, int 'start, int end) returns byte[] {
    byte[] part = [];
    foreach int i in 'start ..< end {
        part.push(bytes[i]);
    }
    return part;
}

function events(http:SseEvent... events) returns stream<http:SseEvent, error?> {
    return new (new Events(events));
}

function eventStreamResponse(string... parts) returns http:Response {
    byte[][] bytes = [];
    foreach string part in parts {
        bytes.push(part.toBytes());
    }
    http:Response res = new;
    res.setByteStream(new stream<byte[], io:Error?>(new Chunks(bytes)), "text/event-stream");
    return res;
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

function printEvents(stream<http:SseEvent, error?> eventStream) returns error? {
    record {|http:SseEvent value;|}? event = check eventStream.next();
    while event is record {|http:SseEvent value;|} {
        io:println(event.value);
        event = check eventStream.next();
    }
}

public function main() returns error? {
    // ---- setSseEventStream / getSseEventStream round trip ----
    http:Response res = new;
    res.setSseEventStream(events({data: "first"}, {event: "update", id: "7", 'retry: 3000, data: "a\nb"},
        {comment: "keep-alive"}));
    io:println(res.getContentType()); // @output text/event-stream
    // As in jBallerina, Cache-Control is only added when the response is sent.
    io:println(res.hasHeader("cache-control")); // @output false
    // @output {"data":"first"}
    // @output {"data":"a\nb","id":"7","retry":3000,"event":"update"}
    // @output {"comment":"keep-alive"}
    check printEvents(check res.getSseEventStream());

    // ---- wire format: field order, one data line per line of data ----
    http:Response wire = new;
    wire.setSseEventStream(events({data: "x\r\ny\rz\n", 'retry: 5, event: "e", id: "1", comment: "c"},
        {data: ""}));
    // @output [": c\nid: 1\nevent: e\nretry: 5\ndata: x\ndata: y\ndata: z\ndata: \n\ndata: \n\n"]
    io:println([check drain(check wire.getByteStream())]);

    // setPayload sends an event stream as text/event-stream whatever content type is passed.
    http:Response viaPayload = new;
    viaPayload.setPayload(events({data: "p"}), "application/json");
    io:println(viaPayload.getContentType()); // @output text/event-stream

    // ---- parsing a text/event-stream body ----
    http:Response raw = eventStreamResponse(":ping\r\n", "data: one\r\n\r\n",
        "event:  spaced\nid:5\nretry: soon\ndata\n\n", "data: crlf\r\rfield-only\n\n",
        "data: line1\ndata:line2\n", "\n", "\n\n", "data: m\n\r\ndata: next\n\n", "unknown: x\ndata: partial");
    // @output {"data":"one","comment":"ping"}
    // @output {"data":"","id":"5","event":" spaced"}
    // @output {"data":"crlf"}
    // @output {}
    // @output {"data":"line1\nline2"}
    // @output {}
    // @output {"data":"m"}
    // @output {"data":"next"}
    check printEvents(check raw.getSseEventStream());

    // A multi-byte character split across reads is decoded once the event is complete.
    byte[] cafe = "data: café\n\n".toBytes();
    http:Response split = new;
    split.setByteStream(new stream<byte[], io:Error?>(new Chunks([bytesBetween(cafe, 0, 10), bytesBetween(cafe, 10, cafe.length())])),
        "text/event-stream; charset=utf-8");
    // @output {"data":"café"}
    check printEvents(check split.getSseEventStream());

    // A closed event stream yields no more events.
    stream<http:SseEvent, error?> closing = check eventStreamResponse("data: a\n\ndata: b\n\n").getSseEventStream();
    io:println(check closing.next()); // @output {"value":{"data":"a"}}
    check closing.close();
    io:println(closing.next() is ()); // @output true

    // An event stream is a valid client target type.
    http:TargetType target = EventStream;
    io:println(target is typedesc<stream<http:SseEvent, error?>>); // @output true
}
