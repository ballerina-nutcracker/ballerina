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

// Services that jBallerina's compiler plugin rejects (TCP_101-TCP_110) are
// rejected at runtime here: a tcp:Service at attach, and a connection service
// when onConnect returns it, which logs the error and closes the connection.
import ballerina/io;
import ballerina/tcp;

service class IntParam {
    *tcp:ConnectionService;

    remote function onBytes(tcp:Caller caller, int x) {
        _ = caller;
        _ = x;
    }
}

service class MutableData {
    *tcp:ConnectionService;

    remote function onBytes(byte[] data) {
        _ = data;
    }
}

service class NoParams {
    *tcp:ConnectionService;

    remote function onBytes() {
    }
}

service class TooManyParams {
    *tcp:ConnectionService;

    remote function onBytes(tcp:Caller caller, readonly & byte[] data, int extra) {
        _ = caller;
        _ = data;
        _ = extra;
    }
}

service class BadReturn {
    *tcp:ConnectionService;

    remote function onBytes(readonly & byte[] data) returns string {
        _ = data;
        return "";
    }
}

service class StringOnError {
    *tcp:ConnectionService;

    remote function onBytes(readonly & byte[] data) {
        _ = data;
    }

    remote function onError(string e) {
        _ = e;
    }
}

service class EmptyOnError {
    *tcp:ConnectionService;

    remote function onBytes(readonly & byte[] data) {
        _ = data;
    }

    remote function onError() {
    }
}

service class OnCloseWithParam {
    *tcp:ConnectionService;

    remote function onBytes(readonly & byte[] data) {
        _ = data;
    }

    remote function onClose(tcp:Caller caller) {
        _ = caller;
    }
}

service class NoOnBytes {
    *tcp:ConnectionService;

    remote function onClose() {
    }
}

service class LocalOnBytes {
    *tcp:ConnectionService;

    function onBytes(readonly & byte[] data) {
        _ = data;
    }
}

service class UnknownRemote {
    *tcp:ConnectionService;

    remote function onBytes(readonly & byte[] data) {
        _ = data;
    }

    remote function onFoo() {
    }
}

int nextService = 0;

function pickConnectionService() returns tcp:ConnectionService {
    nextService += 1;
    match nextService {
        1 => {
            return new IntParam();
        }
        2 => {
            return new MutableData();
        }
        3 => {
            return new NoParams();
        }
        4 => {
            return new TooManyParams();
        }
        5 => {
            return new BadReturn();
        }
        6 => {
            return new StringOnError();
        }
        7 => {
            return new EmptyOnError();
        }
        8 => {
            return new OnCloseWithParam();
        }
        9 => {
            return new NoOnBytes();
        }
        10 => {
            return new LocalOnBytes();
        }
    }
    return new UnknownRemote();
}

service class InvalidPicker {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        _ = caller;
        _ = caller.id;
        return pickConnectionService();
    }
}

service class ExtraRemote {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) {
        _ = caller;
    }

    remote function onBytes(readonly & byte[] data) {
        _ = data;
    }
}

listener tcp:Listener invalidListener = new (19406, localHost = "127.0.0.1");

function init() returns error? {
    check invalidListener.attach(new InvalidPicker());
}

public function testMain() returns error? {
    tcp:Listener other = check new (19407, localHost = "127.0.0.1");
    error? attached = other.attach(new ExtraRemote());
    io:println(attached is error ? attached.message() : "attached"); // @output Function `onBytes` not accepted by the service

    int closed = 0;
    foreach int _ in 0 ..< 11 {
        tcp:Client c = check new ("127.0.0.1", 19406, timeout = 2);
        (readonly & byte[])|tcp:Error r = c->readBytes();
        if r is tcp:Error && r.message() == "Connection closed by the server." {
            closed += 1;
        }
        check c->close();
    }
    io:println(closed); // @output 11
}
