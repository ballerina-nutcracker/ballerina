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

string[] trace = [];

class L {
    string n;

    function init(string n) {
        self.n = n;
        trace.push("listener init " + n);
    }

    public function attach(service object {} svc, () attachPoint = ()) returns error? {
        var _ = svc;
        var _ = attachPoint;
        trace.push("attach " + self.n);
    }

    public function detach(service object {} svc) returns error? {
        var _ = svc;
    }

    public function 'start() returns error? {
    }

    public function gracefulStop() returns error? {
    }

    public function immediateStop() returns error? {
    }
}

int serviceCount = 0;

function nextServiceId() returns int {
    serviceCount += 1;
    return serviceCount;
}

// Both service bodies are identical so the golden does not depend on the
// order in which the two anonymous service classes are dumped.
service on new L("a") {
    function init() {
        trace.push("service init " + nextServiceId().toString());
    }
}

service on new L("b") {
    function init() {
        trace.push("service init " + nextServiceId().toString());
    }
}

public function main() {
    io:println(trace[0]); // @output service init 1
    io:println(trace[1]); // @output listener init a
    io:println(trace[2]); // @output attach a
    io:println(trace[3]); // @output service init 2
    io:println(trace[4]); // @output listener init b
    io:println(trace[5]); // @output attach b
    io:println(trace.length()); // @output 6
}
