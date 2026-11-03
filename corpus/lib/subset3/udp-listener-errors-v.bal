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

service class S {
    *udp:Service;

    remote function onBytes(readonly & byte[] data) {
        _ = data;
    }
}

public function main() returns error? {
    udp:Listener l = check new (41512, {localHost: "127.0.0.1", "foo": 1});
    error? stopBeforeStart = l.gracefulStop();
    io:println(stopBeforeStart is ()); // @output true
    check l.attach(new S());
    error? twice = l.attach(new S());
    io:println(twice is error); // @output true
    error? wrong = l.detach(new S());
    io:println(wrong is error); // @output true
}
