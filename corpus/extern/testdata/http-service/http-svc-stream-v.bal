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

class Repeated {
    private final byte[] block;
    private int remaining;

    isolated function init(int size, int count) {
        byte[] bytes = [];
        foreach int _ in 0 ..< size {
            bytes.push(120);
        }
        self.block = bytes;
        self.remaining = count;
    }

    public isolated function next() returns record {|byte[] value;|}|io:Error? {
        if self.remaining == 0 {
            return;
        }
        self.remaining -= 1;
        return {value: self.block};
    }
}

class Broken {
    private boolean sent = false;

    public isolated function next() returns record {|byte[] value;|}|io:Error? {
        if self.sent {
            return error("disk failure");
        }
        self.sent = true;
        return {value: "partial".toBytes()};
    }
}

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

class FailingEvents {
    private boolean sent = false;

    public isolated function next() returns record {|http:SseEvent value;|}|error? {
        if self.sent {
            return error("db down");
        }
        self.sent = true;
        return {value: {data: "ok"}};
    }
}

class Numbers {
    public isolated function next() returns record {|int value;|}? {
        return;
    }
}

service /streams on new http:Listener(19503) {
    resource function get bytes() returns http:Response {
        http:Response res = new;
        res.setByteStream(new stream<byte[], io:Error?>(new Chunks(["server ", "stream"])));
        return res;
    }

    resource function post count(http:Request req) returns string|error {
        stream<byte[], io:Error?> body = check req.getByteStream(4096);
        int total = 0;
        int blocks = 0;
        record {|byte[] value;|}? block = check body.next();
        while block is record {|byte[] value;|} {
            total += block.value.length();
            blocks += 1;
            block = check body.next();
        }
        return string `bytes=${total} blocks=${blocks}`;
    }

    resource function get events() returns stream<http:SseEvent, error?> {
        return new (new Events([{data: "hello"}, {event: "update", id: "1", data: "a\nb"}]));
    }

    resource function get cached() returns http:Response {
        http:Response res = new;
        res.setHeader("cache-control", "no-store");
        res.setSseEventStream(new stream<http:SseEvent, error?>(new Events([{data: "x"}])));
        return res;
    }

    resource function get failing() returns stream<http:SseEvent, error?> {
        return new (new FailingEvents());
    }

    resource function get numbers() returns stream<int> {
        return new (new Numbers());
    }

    resource function get broken() returns http:Response {
        http:Response res = new;
        res.setByteStream(new stream<byte[], io:Error?>(new Broken()));
        return res;
    }
}

public function testMain() returns error? {
    http:Client c = check new ("http://localhost:19503/streams");

    http:Response bytes = check c->get("/bytes");
    io:println(bytes.getContentType()); // @output application/octet-stream
    io:println(check bytes.getTextPayload()); // @output server stream

    // The body arrives chunked, so the resource reads it from the network as it streams in.
    string counted = check c->post("/count", new stream<byte[], io:Error?>(new Repeated(1000, 10)));
    io:println(counted); // @output bytes=10000 blocks=3

    stream<http:SseEvent, error?> events = check c->get("/events");
    // @output {"data":"hello"}
    // @output {"data":"a\nb","id":"1","event":"update"}
    record {|http:SseEvent value;|}? event = check events.next();
    while event is record {|http:SseEvent value;|} {
        io:println(event.value);
        event = check events.next();
    }

    http:Response raw = check c->get("/events");
    io:println(raw.getContentType()); // @output text/event-stream
    io:println(check raw.getHeader("cache-control")); // @output no-cache,public
    // @output ["data: hello\n\nid: 1\nevent: update\ndata: a\ndata: b\n\n"]
    io:println([check raw.getTextPayload()]);

    http:Response cached = check c->get("/cached");
    io:println(check cached.getHeader("cache-control")); // @output no-store

    // An error from the resource's event stream is sent as a final error event and logged.
    http:Response failing = check c->get("/failing");
    // @output ["data: ok\n\nevent: error\ndata: db down\n\n"]
    io:println([check failing.getTextPayload()]);

    http:Response numbers = check c->get("/numbers");
    io:println(numbers.statusCode); // @output 500

    // An error from a byte-stream body aborts the response instead of ending it cleanly, so the
    // client never sees a complete body.
    boolean incomplete = true;
    http:Response|error broken = c->get("/broken");
    if broken is http:Response {
        incomplete = broken.getTextPayload() is error;
    }
    io:println(incomplete); // @output true
}
