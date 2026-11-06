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

// Exercises Client.writeBytes' write-timeout error path: onConnect holds the
// accepted connection without reading until the client is done, so the
// kernel buffers fill and a write blocks past writeTimeout.
import ballerina/io;
import ballerina/lang.runtime;
import ballerina/tcp;

isolated boolean clientDone = false;

service class IdleConnService {
    *tcp:ConnectionService;
}

service class StalledServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        _ = caller.id;
        foreach int _ in 0 ..< 1000 {
            if isClientDone() {
                break;
            }
            runtime:sleep(0.01);
        }
        return new IdleConnService();
    }
}

listener tcp:Listener stalledListener = new (19409, localHost = "127.0.0.1");

function init() returns error? {
    check stalledListener.attach(new StalledServer());
}

function isClientDone() returns boolean {
    lock {
        return clientDone;
    }
}

public function testMain() returns error? {
    tcp:Client c = check new ("127.0.0.1", 19409, writeTimeout = 0.1);
    byte[] chunk = [];
    chunk[1048575] = 0;
    tcp:Error? writeErr = ();
    foreach int _ in 0 ..< 256 {
        writeErr = c->writeBytes(chunk);
        if writeErr is tcp:Error {
            break;
        }
    }
    lock {
        clientDone = true;
    }
    io:println(writeErr is tcp:Error ? writeErr.message() : "no error"); // @output Write timed out
    check c->close();
    check stalledListener.gracefulStop();
}
