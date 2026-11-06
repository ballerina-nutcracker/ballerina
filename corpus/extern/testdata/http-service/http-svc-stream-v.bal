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
import ballerina/lang.runtime;

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

class PanickingEvents {
    private boolean sent = false;

    public isolated function next() returns record {|http:SseEvent value;|}|error? {
        if self.sent {
            int[] empty = [];
            return {value: {data: empty[5].toString()}};
        }
        self.sent = true;
        return {value: {data: "ok"}};
    }
}

class Ticks {
    private final int count;
    private final decimal gap;
    private int sent = 0;

    isolated function init(int count, decimal gap) {
        self.count = count;
        self.gap = gap;
    }

    public isolated function next() returns record {|http:SseEvent value;|}|error? {
        if self.sent >= self.count {
            return;
        }
        if self.sent > 0 {
            runtime:sleep(self.gap);
        }
        self.sent += 1;
        return {value: {data: self.sent.toString()}};
    }
}

class Numbers {
    public isolated function next() returns record {|int value;|}? {
        return;
    }
}

isolated int blocksPulled = 0;

class CountedBlocks {
    private int remaining = 1000;

    public isolated function next() returns record {|byte[] value;|}|io:Error? {
        if self.remaining == 0 {
            return;
        }
        self.remaining -= 1;
        lock {
            blocksPulled += 1;
        }
        return {value: "block".toBytes()};
    }
}

final http:Client backend = check new ("http://localhost:19503/streams");

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

    resource function get panicking() returns stream<http:SseEvent, error?> {
        return new (new PanickingEvents());
    }

    resource function get ticks() returns stream<http:SseEvent, error?> {
        return new (new Ticks(5, 0.2));
    }

    resource function get stalled() returns stream<http:SseEvent, error?> {
        return new (new Ticks(2, 1.5));
    }

    resource function get numbers() returns stream<int> {
        return new (new Numbers());
    }

    resource function get broken() returns http:Response {
        http:Response res = new;
        res.setByteStream(new stream<byte[], io:Error?>(new Broken()));
        return res;
    }

    resource function head counted() returns http:Response {
        http:Response res = new;
        res.setByteStream(new stream<byte[], io:Error?>(new CountedBlocks()));
        return res;
    }

    resource function get pulled() returns int {
        lock {
            return blocksPulled;
        }
    }

    resource function get hello() returns string => "hello";

    resource function get digits() returns string {
        string digits = "";
        foreach int _ in 0 ..< 2000 {
            digits += "0123456789";
        }
        return digits;
    }

    // The backend response carries Content-Length: 5, which the longer stream replaces.
    resource function get replaced() returns http:Response|error {
        http:Response res = check backend->get("/hello");
        res.setByteStream(new stream<byte[], io:Error?>(new Chunks(["replaced ", "by a stream"])));
        return res;
    }

    resource function get remainder() returns http:Response|error {
        http:Response res = check backend->get("/digits");
        stream<byte[], io:Error?> body = check res.getByteStream(5);
        _ = check body.next();
        return res;
    }

    resource function post echo(http:Request req) returns http:Response|error {
        http:Response res = new;
        res.setByteStream(check req.getByteStream());
        return res;
    }

    resource function post relay(http:Request req) returns string|error {
        stream<byte[], io:Error?> body = check req.getByteStream(5);
        _ = check body.next();
        string counted = check backend->post("/count", req);
        return counted;
    }

    resource function post redirect() returns http:Response {
        http:Response res = new;
        res.statusCode = 307;
        res.setHeader("location", "/streams/count");
        return res;
    }

    resource function get multiline() returns stream<http:SseEvent, error?> {
        return new (new Events([{comment: "first\n\ndata: forged", data: "real"}, {id: "1\n\ndata: forged", data: "x"}]));
    }
}

service /quick on new http:Listener(19504, {timeout: 0.5}) {
    // Each gap between events is longer than the listener timeout, which applies to writes only.
    resource function get ticks() returns stream<http:SseEvent, error?> {
        return new (new Ticks(3, 0.8));
    }

    resource function get relay() returns http:Response|error {
        return backend->get("/ticks");
    }

    resource function get relayRest() returns http:Response|error {
        http:Response res = check backend->get("/ticks");
        stream<byte[], io:Error?> body = check res.getByteStream(1);
        _ = check body.next();
        return res;
    }
}

public function testMain() returns error? {
    http:Client c = check new ("http://localhost:19503/streams");
    http:Client h1 = check new ("http://localhost:19503/streams", {httpVersion: http:HTTP_1_1});

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

    // Over HTTP/1.1 an event stream also carries Connection: keep-alive; other responses do not.
    http:Response h1Events = check h1->get("/events");
    io:println(h1Events.getHeader("connection")); // @output keep-alive
    _ = check h1Events.getTextPayload();
    http:Response h1Bytes = check h1->get("/bytes");
    io:println(h1Bytes.hasHeader("connection")); // @output false
    _ = check h1Bytes.getTextPayload();

    http:Response cached = check c->get("/cached");
    io:println(check cached.getHeader("cache-control")); // @output no-store

    // An error from the resource's event stream is sent as a final error event and logged.
    http:Response failing = check c->get("/failing");
    // @output ["data: ok\n\nevent: error\ndata: db down\n\n"]
    io:println([check failing.getTextPayload()]);

    // A panic in the resource's event stream after the headers were sent aborts the response
    // instead of appending an error body to the event stream.
    stream<http:SseEvent, error?> panicking = check h1->get("/panicking");
    io:println(check panicking.next()); // @output {"value":{"data":"ok"}}
    io:println(panicking.next() is error); // @output true

    // The client timeout is an idle timeout: a stream that keeps delivering outlives it, and
    // one that stalls for longer fails.
    http:Client impatient = check new ("http://localhost:19503/streams", {timeout: 0.5});
    stream<http:SseEvent, error?> ticks = check impatient->get("/ticks");
    int received = 0;
    record {|http:SseEvent value;|}? tick = check ticks.next();
    while tick is record {|http:SseEvent value;|} {
        received += 1;
        tick = check ticks.next();
    }
    io:println(received); // @output 5

    stream<http:SseEvent, error?> stalled = check impatient->get("/stalled");
    io:println(check stalled.next()); // @output {"value":{"data":"1"}}
    record {|http:SseEvent value;|}|error? afterStall = stalled.next();
    // @output Error occurred while reading stream: Idle timeout triggered while reading inbound response entity body
    io:println(afterStall is error ? afterStall.message() : "no timeout");

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

    // A HEAD response never reads its byte-stream body, which would only be discarded.
    http:Response head = check h1->head("/counted");
    io:println(head.statusCode); // @output 200
    int pulled = check c->get("/pulled");
    io:println(pulled); // @output 0

    // A stream that replaces a backend body drops the backend's Content-Length.
    string replaced = check h1->get("/replaced");
    io:println(replaced); // @output replaced by a stream

    // A partly read body sends on the part getByteStream has not read.
    string remainder = check h1->get("/remainder");
    io:println(remainder.length(), " ", remainder.substring(0, 10)); // @output 19995 5678901234
    string digits = check c->get("/digits");
    string relayed = check h1->post("/relay", digits);
    io:println(relayed); // @output bytes=19995 blocks=5

    // A response can stream the body of the request it answers, over HTTP/1.1 as well.
    byte[] echoed = check h1->post("/echo", new stream<byte[], io:Error?>(new Repeated(1000, 100)));
    io:println(echoed.length()); // @output 100000

    // A byte-stream body cannot be replayed, so a 307 answering it is returned, not followed.
    http:Client following = check new ("http://localhost:19503/streams", {followRedirects: {enabled: true}});
    http:Response redirected = check following->post("/redirect", new stream<byte[], io:Error?>(new Chunks(["a"])));
    io:println(redirected.statusCode); // @output 307
    string followed = check following->post("/redirect", "text body");
    io:println(followed); // @output bytes=9 blocks=1

    // Each comment line gets its own prefix, and an id or event name holding a line break is
    // rejected, so no value can end its event early and forge another.
    http:Response multiline = check c->get("/multiline");
    // @output [": first\n: \n: data: forged\ndata: real\n\nevent: error\ndata: Invalid value provided as event: the id field must not contain a line break.\n\n"]
    io:println([check multiline.getTextPayload()]);

    // A gap between events longer than the listener timeout does not reset an HTTP/2 stream.
    http:Client quick = check new ("http://localhost:19504/quick");
    stream<http:SseEvent, error?> quickTicks = check quick->get("/ticks");
    int quickReceived = 0;
    record {|http:SseEvent value;|}? quickTick = check quickTicks.next();
    while quickTick is record {|http:SseEvent value;|} {
        quickReceived += 1;
        quickTick = check quickTicks.next();
    }
    io:println(quickReceived); // @output 3

    // A relayed body that keeps arriving outlives the listener timeout, whether or not it was partly read.
    string relayedTicks = check quick->get("/relay");
    io:println(relayedTicks.length()); // @output 45
    string relayedRest = check quick->get("/relayRest");
    io:println(relayedRest.length()); // @output 44
}
