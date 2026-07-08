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

public type CounterService isolated service object {
    isolated function inc();
    isolated function get() returns int;
};

class SimpleListener {
    private CounterService? s = ();

    public function attach(CounterService svc, () attachPoint = ()) returns () {
        var _ = svc;
        var _ = attachPoint;
        self.s = svc;
    }

    public function detach(CounterService svc) returns error? {
        var _ = svc;
        self.s = ();
    }

    public function 'start() returns error? {
    }

    public function gracefulStop() returns error? {
    }

    public function immediateStop() returns error? {
    }

    function inc() {
        var svc = self.s;
        if svc == () {
            panic error("no service");
        }
        svc.inc();
    }

    function get() returns int {
        var svc = self.s;
        if svc == () {
            panic error("no service");
        }
        return svc.get();
    }
}

listener SimpleListener l = new ();

isolated service on l {
    private int count = 0;

    isolated function inc() {
        lock {
            self.count = self.count + 1;
        }
    }

    isolated function get() returns int {
        lock {
            return self.count;
        }
    }
}

public function main() {
    l.inc();
    l.inc();
    l.inc();
    io:println(l.get()); // @output 3
}
