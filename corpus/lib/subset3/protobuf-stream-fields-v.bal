// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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
import ballerina/protobuf.types.'any as pbany;
import ballerina/protobuf.types.duration;
import ballerina/protobuf.types.empty;
import ballerina/protobuf.types.'struct;
import ballerina/protobuf.types.timestamp;
import ballerina/protobuf.types.wrappers;

class IntCounter {
    int n = 0;

    public isolated function next() returns record {| int value; |}|() {
        int current = self.n;
        self.n = current + 1;
        return {value: current};
    }

    public isolated function close() returns () {
        return ();
    }
}

// Proves stream<T, error?> record fields declared across every protobuf submodule
// (Any/Duration/Struct/Timestamp/wrapper Context*Stream types) type-check, without
// needing a matching iterator per element type.
function acceptStreamFields(
        pbany:ContextAnyStream anyStream, duration:ContextDurationStream durationStream,
        'struct:ContextStructStream structStream, timestamp:ContextTimestampStream timestampStream,
        wrappers:ContextBooleanStream booleanStream, wrappers:ContextBytesStream bytesStream,
        wrappers:ContextFloatStream floatStream, wrappers:ContextStringStream stringStream) {
    _ = anyStream;
    _ = durationStream;
    _ = structStream;
    _ = timestampStream;
    _ = booleanStream;
    _ = bytesStream;
    _ = floatStream;
    _ = stringStream;
}

public function main() {
    stream<int, error?> s = new (new IntCounter());
    wrappers:ContextIntStream intStream = {content: s, headers: {}};
    record {| int value; |}|error? r = intStream.content.next();
    if r is record {| int value; |} {
        io:println(r.value); // @output 0
    }

    empty:ContextNil nilCtx = {headers: {}};
    io:println(nilCtx.headers.length()); // @output 0

    io:println(acceptStreamFields is function); // @output true
}
