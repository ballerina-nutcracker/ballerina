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

// A connection closed before onConnect returns still gets exactly one onClose
// on the service onConnect returns, whether onConnect closes the Caller itself
// or a write from onConnect finds the peer gone. jBallerina never dispatches
// onClose in either case (ballerina-library#9407).
import ballerina/io;
import ballerina/lang.runtime;
import ballerina/tcp;

isolated int localCloses = 0;
isolated int peerCloses = 0;

service class TrackedConnService {
    *tcp:ConnectionService;
    private final boolean local;

    isolated function init(boolean local) {
        self.local = local;
    }

    remote function onBytes(readonly & byte[] data) {
        _ = data;
    }

    remote function onClose() {
        if self.local {
            lock {
                localCloses += 1;
            }
        } else {
            lock {
                peerCloses += 1;
            }
        }
    }
}

service class LocalCloseServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService|tcp:Error {
        check caller->close();
        check caller->close();
        return new TrackedConnService(true);
    }
}

service class PeerCloseServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        foreach int _ in 0 ..< 500 {
            tcp:Error? err = caller->writeBytes("hi".toBytes());
            if err is tcp:Error {
                break;
            }
            runtime:sleep(0.01);
        }
        return new TrackedConnService(false);
    }
}

listener tcp:Listener localCloseListener = new (19414, localHost = "127.0.0.1");
listener tcp:Listener peerCloseListener = new (19415, localHost = "127.0.0.1");

function init() returns error? {
    check localCloseListener.attach(new LocalCloseServer());
    check peerCloseListener.attach(new PeerCloseServer());
}

function closeCount(boolean local) returns int {
    if local {
        lock {
            return localCloses;
        }
    }
    lock {
        return peerCloses;
    }
}

function waitForClose(boolean local) {
    foreach int _ in 0 ..< 500 {
        if closeCount(local) > 0 {
            break;
        }
        runtime:sleep(0.01);
    }
    runtime:sleep(0.2);
}

public function testMain() returns error? {
    tcp:Client c1 = check new ("127.0.0.1", 19414, timeout = 2);
    readonly & byte[]|tcp:Error response = c1->readBytes();
    io:println("client read failed: ", response is tcp:Error); // @output client read failed: true
    check c1->close();
    waitForClose(true);
    io:println("local onClose count: ", closeCount(true)); // @output local onClose count: 1

    tcp:Client c2 = check new ("127.0.0.1", 19415, timeout = 2);
    check c2->close();
    waitForClose(false);
    io:println("peer onClose count: ", closeCount(false)); // @output peer onClose count: 1

    check localCloseListener.gracefulStop();
    check peerCloseListener.gracefulStop();
}
