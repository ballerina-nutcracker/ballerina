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

class Big {
    private int n = 0;

    public isolated function next() returns record {|byte[] value;|}|io:Error? {
        if self.n == 1000 {
            return;
        }
        self.n += 1;
        byte[] b = [];
        b[65535] = 1;
        return {value: b};
    }
}

class Paced {
    private int sent = 0;

    public isolated function next() returns record {|byte[] value;|}|io:Error? {
        if self.sent == 5 {
            return;
        }
        self.sent += 1;
        runtime:sleep(0.05);
        return {value: "0123456789".toBytes()};
    }
}

class Panicking {
    private boolean sent = false;

    public isolated function next() returns record {|byte[] value;|}|io:Error? {
        if self.sent {
            int[] empty = [];
            return {value: [<byte>empty[3]]};
        }
        self.sent = true;
        return {value: "abc".toBytes()};
    }
}

function postSink(http:Client c, stream<byte[], io:Error?> body) returns string|error {
    string response = check c->post("/sink", body);
    return response;
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

    // The whole body is sent even when the server answers before it has read it.
    string early = check c->post("/early", new stream<byte[], io:Error?>(new Paced()));
    io:println(early); // @output read=50

    // An error response ends the upload instead of waiting for the server to read the rest.
    http:Client unbounded = check new ("http://testserver", {timeout: 0});
    http:Response rejected = check unbounded->post("/reject", new stream<byte[], io:Error?>(new Big()));
    io:println(rejected.statusCode, " ", (check rejected.getTextPayload()).length()); // @output 413 65536
    string rejectResult = check c->get("/reject-result");
    io:println(rejectResult); // @output upload stopped

    // A panic in the body stream's next() ends the request instead of leaving it open.
    string|error panicked = trap postSink(c, new (new Panicking()));
    io:println(panicked is error ? panicked.message() : panicked); // @output invalid array index: 3
    string sink = check c->get("/sink-result");
    io:println(sink); // @output read=3 failed=true

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

    // A body cut short before its Content-Length ends its byte stream with the read error.
    http:Response truncated = check c->get("/truncated");
    io:println(blocks(check truncated.getByteStream())); // @output error GenericError ("Error occurred while reading stream:unexpected EOF")

    // A byte order mark at the start of an event stream is dropped, even when split across reads.
    stream<http:SseEvent, error?> bom = check c->get("/events-bom");
    io:println(collectEvents(bom)); // @output [{"data":"hi"}]

    // The media type is compared case-insensitively and without its parameters.
    stream<http:SseEvent, error?> mixedCase = check c->get("/events-mixed-case");
    io:println(collectEvents(mixedCase)); // @output [{"data":"mixed"}]
    stream<http:SseEvent, error?>|error extra = c->get("/events-extra");
    io:println(extra is error ? extra.message() : "stream"); // @output invalid payload target type. The response is not of text/event-stream content type.

    // A second event stream over the same response carries on where the first left off.
    http:Response burst = check c->get("/events-burst");
    stream<http:SseEvent, error?> firstEvents = check burst.getSseEventStream();
    io:println(check firstEvents.next()); // @output {"value":{"data":"one"}}
    io:println(collectEvents(check burst.getSseEventStream())); // @output [{"data":"two"},{"data":"three"}]

    // A narrower element type could not hold every event, such as a comment-only one.
    stream<record {|string data;|}, error?>|error narrow = c->get("/events");
    // @output error PayloadBindingClientError ("payload binding failed: Target return type must accept stream<http:SseEvent, error?>")
    io:println(narrow);
}

function collectEvents(stream<http:SseEvent, error?> eventStream) returns http:SseEvent[]|error {
    http:SseEvent[] received = [];
    record {|http:SseEvent value;|}? event = check eventStream.next();
    while event is record {|http:SseEvent value;|} {
        received.push(event.value);
        event = check eventStream.next();
    }
    return received;
}
