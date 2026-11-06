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
        io:println("start ", self.n);
    }

    public function gracefulStop() returns error? {
    }

    public function immediateStop() returns error? {
    }
}

service on new L("one"), new L("two") {
    function init() {
        trace.push("service init");
    }
}

public function main() {
    io:println(trace[0]); // @output service init
    io:println(trace[1]); // @output listener init one
    io:println(trace[2]); // @output attach one
    io:println(trace[3]); // @output listener init two
    io:println(trace[4]); // @output attach two
    io:println(trace.length()); // @output 5
    // @output start one
    // @output start two
}
