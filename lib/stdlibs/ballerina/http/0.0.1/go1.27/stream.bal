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

import ballerina/io;
import ballerina/log;

# Represents a Server Sent Event emitted from a service.
public type SseEvent record {|
    # Name of the event
    string event?;
    # Id of the event
    string id?;
    # Data part of the event
    string data?;
    # The reconnect time on failure in milliseconds.
    int 'retry?;
    # Comment added to the event
    string comment?;
|};

isolated function getByteStream(Request|Response message, int arraySize) returns stream<byte[], io:Error?>|error {
    stream<byte[], io:Error?>|error? byteStream = externGetByteStream(message, arraySize);
    if byteStream is () {
        stream<byte[], io:Error?> bodyStream = new (new BodyByteStream(message, arraySize));
        return bodyStream;
    }
    return byteStream;
}

// Reads a request or response body received from the network.
class BodyByteStream {
    private final Request|Response message;
    private final int arraySize;
    private boolean isClosed = false;

    isolated function init(Request|Response message, int arraySize) {
        self.message = message;
        self.arraySize = arraySize;
    }

    public isolated function next() returns record {|byte[] value;|}|io:Error? {
        byte[]|io:Error? bytes = externReadBodyBytes(self.message, self.arraySize);
        return bytes is byte[] ? {value: bytes} : bytes;
    }

    public isolated function close() returns io:Error? {
        if !self.isClosed {
            self.isClosed = true;
            return externCloseBody(self.message);
        }
    }
}

isolated function getSseEventStream(Response response) returns stream<SseEvent, error?>|error {
    check externPrepareSseEventStream(response);
    stream<SseEvent, error?> eventStream = new (new BytesToEventStreamGenerator(response));
    return eventStream;
}

// Turns a `text/event-stream` response body into `SseEvent` records, handing each event on as
// soon as it is complete.
class BytesToEventStreamGenerator {
    private final Response response;
    private boolean isClosed = false;

    isolated function init(Response response) {
        self.response = response;
    }

    public isolated function next() returns record {|SseEvent value;|}|error? {
        if self.isClosed {
            return;
        }
        byte[]|error? eventBytes = externReadSseEvent(self.response);
        if eventBytes is () {
            return;
        }
        if eventBytes is error {
            return sseEventError(eventBytes);
        }
        string|error eventText = string:fromBytes(eventBytes);
        if eventText is error {
            return sseEventError(eventText);
        }
        SseEvent event = {};
        externParseSseEvent(eventText, event);
        return {value: event};
    }

    public isolated function close() returns error? {
        self.isClosed = true;
        return externCloseBody(self.response);
    }
}

isolated function sseEventError(error err) returns error {
    log:printError("failed to construct SseEvent", err);
    return err;
}

// Serialises a stream of `SseEvent` records into the `text/event-stream` wire format. An error
// from the event stream, or an event with no fields, is sent as a final `error` event.
class SseEventToByteStreamGenerator {
    private final stream<SseEvent, error?> eventStream;
    private boolean isClosed = false;
    private boolean isErrorOccurred = false;

    isolated function init(stream<SseEvent, error?> eventStream) {
        self.eventStream = eventStream;
    }

    public isolated function next() returns record {|byte[] value;|}|io:Error? {
        if self.isClosed || self.isErrorOccurred {
            return;
        }
        record {|SseEvent value;|}|error? event = self.eventStream.next();
        if event is () {
            return;
        }
        if event is error {
            return self.errorEvent(event);
        }
        error? invalidEvent = validateSseEvent(event.value);
        if invalidEvent is error {
            return self.errorEvent(invalidEvent);
        }
        return {value: externGetSseEventBytes(event.value)};
    }

    public isolated function close() returns error? {
        check self.eventStream.close();
        self.isClosed = true;
    }

    private isolated function errorEvent(error err) returns record {|byte[] value;|} {
        self.isErrorOccurred = true;
        log:printError("unable to obtain byte array", err);
        return {value: externGetSseEventBytes({event: "error", data: err.message()})};
    }
}

isolated function validateSseEvent(SseEvent event) returns error? {
    if !event.hasKey("event") && !event.hasKey("id") && !event.hasKey("retry")
            && !event.hasKey("comment") && !event.hasKey("data") {
        return error("Invalid value provided as event: "
            + "at least one field is expected to be present in the SseEvent record.");
    }
}

// Called by the listener for a resource function that returns a stream.
isolated function createSseResponse(any message) returns Response|error {
    if message is stream<SseEvent, error?>|stream<SseEvent, error> {
        Response response = new;
        response.setSseEventStream(message);
        return response;
    }
    return error("invalid response body type. expected one of the types: anydata|http:StatusCodeResponse|http:Response");
}

// Called by the client when the target type is neither `http:Response` nor `anydata`.
isolated function isEventStreamTarget(TargetType targetType) returns boolean {
    return targetType is typedesc<stream<SseEvent, error?>>;
}

isolated function externSetNoCache(Response response) = external;

isolated function externGetByteStream(Request|Response message, int arraySize)
        returns stream<byte[], io:Error?>|error? = external;

isolated function externReadBodyBytes(Request|Response message, int arraySize) returns byte[]|io:Error? = external;

isolated function externCloseBody(Request|Response message) returns io:Error? = external;

isolated function externPrepareSseEventStream(Response response) returns error? = external;

isolated function externReadSseEvent(Response response) returns byte[]|error? = external;

isolated function externParseSseEvent(string event, SseEvent target) = external;

isolated function externGetSseEventBytes(SseEvent event) returns byte[] = external;
