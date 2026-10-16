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
import ballerina/udp;

service class ExtraParam {
    *udp:Service;

    remote function onBytes(readonly & byte[] data, udp:Caller caller, int extra) {
        _ = data;
        _ = caller;
        _ = extra;
    }
}

service class IntData {
    *udp:Service;

    remote function onBytes(int data) {
        _ = data;
    }
}

service class MutableBytes {
    *udp:Service;

    remote function onBytes(byte[] data) {
        _ = data;
    }
}

service class BytesForDatagram {
    *udp:Service;

    remote function onDatagram(readonly & byte[] data) {
        _ = data;
    }
}

service class NoParams {
    *udp:Service;

    remote function onBytes() {
    }
}

service class StringError {
    *udp:Service;

    remote function onBytes(readonly & byte[] data) {
        _ = data;
    }

    remote function onError(string e) {
        _ = e;
    }
}

service class EmptyError {
    *udp:Service;

    remote function onBytes(readonly & byte[] data) {
        _ = data;
    }

    remote function onError() {
    }
}

service class NotRemote {
    *udp:Service;

    function onBytes(readonly & byte[] data) {
        _ = data;
    }
}

service class NoHandler {
    *udp:Service;
}

service class BothHandlers {
    *udp:Service;

    remote function onBytes(readonly & byte[] data) {
        _ = data;
    }

    remote function onDatagram(readonly & udp:Datagram datagram) {
        _ = datagram;
    }
}

service class UnknownRemote {
    *udp:Service;

    remote function onBytes(readonly & byte[] data) {
        _ = data;
    }

    remote function onFoo() {
    }
}

service class StringReturn {
    *udp:Service;

    remote function onBytes(readonly & byte[] data) returns string {
        _ = data;
        return "";
    }
}

service class IntErrorReturn {
    *udp:Service;

    remote function onBytes(readonly & byte[] data) {
        _ = data;
    }

    remote function onError(udp:Error err) returns int {
        _ = err.message();
        return 0;
    }
}

service class Valid {
    *udp:Service;

    remote function onDatagram(udp:Caller caller, readonly & udp:Datagram datagram) returns byte[]|udp:Datagram|udp:Error? {
        _ = caller;
        return datagram.data;
    }

    remote function onError(readonly & udp:Error err) returns udp:Error? {
        return err;
    }
}

function attachMessage(udp:Listener l, udp:Service s) returns string {
    error? result = l.attach(s);
    return result is error ? result.message() : "attached";
}

public function main() returns error? {
    udp:Listener l = check new (41513, {localHost: "127.0.0.1"});
    io:println(attachMessage(l, new ExtraParam())); // @output Provided 3 parameters, `onBytes` can have only 2 parameters
    io:println(attachMessage(l, new IntData())); // @output Invalid parameter `data` provided for `onBytes`, function expects `readonly & byte[]`.
    io:println(attachMessage(l, new MutableBytes())); // @output Invalid parameter `data` provided for `onBytes`, function expects `readonly & byte[]`.
    io:println(attachMessage(l, new BytesForDatagram())); // @output Invalid parameter `data` provided for `onDatagram`, function expects `readonly & udp:Datagram`.
    io:println(attachMessage(l, new NoParams())); // @output No parameter provided for `onBytes`, function expects `readonly & byte[]` as a parameter.
    io:println(attachMessage(l, new StringError())); // @output Invalid parameter `e` provided for `onError`, function expects `udp:Error`.
    io:println(attachMessage(l, new EmptyError())); // @output No parameter provided for `onError`, function expects `udp:Error` as a parameter.
    io:println(attachMessage(l, new NotRemote())); // @output `remote` keyword expected in `onBytes` function signature.
    io:println(attachMessage(l, new NoHandler())); // @output Service does not contain `onDatagram` or `onBytes` function.
    io:println(attachMessage(l, new BothHandlers())); // @output Service cannot contain both `onDatagram` and `onBytes` functions.
    io:println(attachMessage(l, new UnknownRemote())); // @output Function `onFoo` not accepted by the service
    io:println(attachMessage(l, new StringReturn())); // @output Invalid return type provided for function `onBytes`, return type should be a subtype of `byte[] | udp:Datagram | udp:Error?`
    io:println(attachMessage(l, new IntErrorReturn())); // @output Invalid return type provided for function `onError`, return type should be a subtype of `udp:Error | ()`
    io:println(attachMessage(l, new Valid())); // @output attached
}
