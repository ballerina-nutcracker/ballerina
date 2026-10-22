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

function chunks(string... parts) returns stream<byte[], io:Error?> {
    return new (new Chunks(parts));
}

function drain(stream<byte[], io:Error?> byteStream) returns string|error {
    string text = "";
    record {|byte[] value;|}? block = check byteStream.next();
    while block is record {|byte[] value;|} {
        text += check string:fromBytes(block.value) + "|";
        block = check byteStream.next();
    }
    return text;
}

public function main() returns error? {
    // ---- Request.setByteStream / getByteStream ----
    http:Request req = new;
    req.setByteStream(chunks("ab", "cd"));
    io:println(req.getContentType()); // @output application/octet-stream
    io:println(drain(check req.getByteStream())); // @output ab|cd|

    http:Request reqTyped = new;
    reqTyped.setByteStream(chunks("x"), "text/csv");
    io:println(reqTyped.getContentType()); // @output text/csv

    http:Request reqExisting = new;
    check reqExisting.setContentType("application/pdf");
    reqExisting.setByteStream(chunks("x"));
    io:println(reqExisting.getContentType()); // @output application/pdf

    // ---- Response.setByteStream / getByteStream ----
    http:Response res = new;
    res.setByteStream(chunks("one", "two", "three"), "application/x-ndjson");
    io:println(res.getContentType()); // @output application/x-ndjson
    io:println(drain(check res.getByteStream(2))); // @output one|two|three|

    // As in jBallerina, the binary payload getter sees a body set with setByteStream as empty.
    http:Response resBinary = new;
    resBinary.setByteStream(chunks("abc", "de"));
    byte[] binary = check resBinary.getBinaryPayload();
    io:println(binary.length()); // @output 0

    // ---- setPayload byte-stream arms ----
    http:Request reqPayload = new;
    reqPayload.setPayload(chunks("p", "q"));
    io:println(reqPayload.getContentType()); // @output application/octet-stream
    io:println(drain(check reqPayload.getByteStream())); // @output p|q|

    http:Response resPayload = new;
    resPayload.setPayload(chunks("r"), "text/plain");
    io:println(resPayload.getContentType()); // @output text/plain
    io:println(drain(check resPayload.getByteStream())); // @output r|

    // A byte stream is a valid client request message.
    http:RequestMessage message = chunks("m");
    io:println(message is stream<byte[], io:Error?>); // @output true
}
