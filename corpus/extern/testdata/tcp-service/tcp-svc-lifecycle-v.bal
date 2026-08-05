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

// Exercises Listener.detach (identity-checked, unlike jBallerina's
// any-detach-clears-whatever's-attached bug) and Listener.immediateStop
// (a real force-stop in this port, unlike jBallerina's unimplemented stub —
// see the README's Notable Behavioural Changes).
import ballerina/io;
import ballerina/tcp;

service class NoopConnService {
    *tcp:ConnectionService;
}

service class NoopServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        _ = caller.id;
        return new NoopConnService();
    }
}

listener tcp:Listener lifecycleListener = new (19392);

function init() returns error? {
    NoopServer first = new NoopServer();
    check lifecycleListener.attach(first);
    check lifecycleListener.detach(first);
    check lifecycleListener.attach(new NoopServer());
}

public function testMain() returns error? {
    tcp:Client c = check new ("localhost", 19392, {});
    check c->close();

    check lifecycleListener.immediateStop();
    tcp:Client|tcp:Error c2 = new ("localhost", 19392, {});
    io:println(c2 is tcp:Error); // @output true
}
