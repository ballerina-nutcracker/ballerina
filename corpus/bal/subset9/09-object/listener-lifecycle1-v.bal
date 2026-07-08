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

class LifecycleListener {
    function init() {
        io:println("listener init"); // @output listener init
    }

    public function attach(service object {} svc, () attachPoint = ()) returns () {
        var _ = svc;
        var _ = attachPoint;
    }

    public function detach(service object {} svc) returns error? {
        var _ = svc;
    }

    public function 'start() returns error? {
        io:println("listener start");
    }

    public function gracefulStop() returns error? {
        io:println("listener graceful stop");
    }

    public function immediateStop() returns error? {
    }
}

public listener LifecycleListener l = new ();

public function main() {
    io:println("main"); // @output main
}

// $start (lifecycle hook) runs after main, before the harness invokes
// testMain on the parked runtime.
// @output listener start

// testMain is invoked by the test harness after Listen() returns, while the
// runtime is parked in StateListening. This validates that the harness's
// post-listen hook fires between $start and $gracefulStop.
public function testMain() {
    io:println("testMain"); // @output testMain
}

// $gracefulStop runs once the harness pushes a graceful stop signal.
// @output listener graceful stop
