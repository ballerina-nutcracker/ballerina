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

function makeListener(string n) returns L {
    trace.push("make " + n);
    return new L(n);
}

listener L declared = new ("declared");

service on new L("inline"), declared, makeListener("call") {
    function init() {
        trace.push("service init");
    }
}

public function main() {
    io:println(trace[0]); // @output listener init declared
    io:println(trace[1]); // @output service init
    io:println(trace[2]); // @output listener init inline
    io:println(trace[3]); // @output attach inline
    io:println(trace[4]); // @output attach declared
    io:println(trace[5]); // @output make call
    io:println(trace[6]); // @output listener init call
    io:println(trace[7]); // @output attach call
    io:println(trace.length()); // @output 8
}
