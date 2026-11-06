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

// Exercises Listener.detach (identity-checked, unlike jBallerina),
// Listener.immediateStop (a real force-stop in this port, unlike jBallerina's
// unimplemented stub), and restarting a stopped listener; see the README's
// Notable Behavioural Changes.
import ballerina/io;
import ballerina/tcp;

service class EchoConnService {
    *tcp:ConnectionService;

    remote function onBytes(readonly & byte[] data) returns byte[] {
        return data;
    }
}

service class EchoServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        _ = caller.id;
        return new EchoConnService();
    }
}

listener tcp:Listener lifecycleListener = new (19392);

function init() returns error? {
    EchoServer first = new EchoServer();
    check lifecycleListener.attach(first);
    check lifecycleListener.detach(first);
    check lifecycleListener.attach(new EchoServer());
}

public function testMain() returns error? {
    tcp:Client c = check new ("localhost", 19392, timeout = 2);
    check c->writeBytes("ping".toBytes());
    readonly & byte[] echoed = check c->readBytes();
    io:println('string:fromBytes(echoed)); // @output ping

    // gracefulStop leaves the open connection up; a following immediateStop must still close it.
    check lifecycleListener.gracefulStop();
    check lifecycleListener.immediateStop();
    (readonly & byte[])|tcp:Error afterStop = c->readBytes();
    io:println(afterStop is tcp:Error ? afterStop.message() : "data"); // @output Connection closed by the server.
    check c->close();
    tcp:Client|tcp:Error c2 = new ("localhost", 19392);
    io:println(c2 is tcp:Error); // @output true

    tcp:Listener bare = check new (19402, localHost = "127.0.0.1");
    check bare.'start();
    tcp:Client c3 = check new ("127.0.0.1", 19402, timeout = 2);
    io:println("connected without a service"); // @output connected without a service
    (readonly & byte[])|tcp:Error r3 = c3->readBytes();
    io:println(r3 is tcp:Error ? r3.message() : "data"); // @output Connection closed by the server.
    check bare.immediateStop();

    tcp:Listener restartable = check new (19405, localHost = "127.0.0.1");
    check restartable.attach(new EchoServer());
    check restartable.'start();
    error? secondStart = restartable.'start();
    io:println(secondStart is error ? secondStart.message() : "started twice"); // @output Error initializing the server: Address already in use
    check restartable.gracefulStop();
    check restartable.'start();
    tcp:Client c4 = check new ("127.0.0.1", 19405, timeout = 2);
    check c4->writeBytes("pong".toBytes());
    readonly & byte[] restarted = check c4->readBytes();
    io:println('string:fromBytes(restarted)); // @output pong
    check c4->close();
    check restartable.immediateStop();
}
