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

public type ServiceLifeCycle service object {
    public function onAttach(string message);
    public function onDetach(string message);
    function trigger(string message);
};

public class MyListener {
    private ServiceLifeCycle? s = ();

    public function attach(ServiceLifeCycle svc, () attachPoint = ()) returns () {
        var _ = svc;
        var _ = attachPoint;
        svc.onAttach("attached");
        self.s = svc;
    }

    public function detach(ServiceLifeCycle svc) returns error? {
        var _ = svc;
        svc.onDetach("detached");
        self.s = ();
    }

    public function 'start() returns error? {
        io:println("listener start");
    }

    public function gracefulStop() returns error? {
        io:println("graceful stop");
    }

    public function immediateStop() returns error? {
        io:println("immediate stop");
    }

    function trigger(string message) {
        var svc = self.s;
        if svc == () {
            panic error("no service");
        }
        svc.trigger(message);
    }
}
