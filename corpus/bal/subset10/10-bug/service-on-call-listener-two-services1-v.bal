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

class L {
    string n;

    function init(string n) {
        self.n = n;
        io:println("listener init ", n);
    }

    public function attach(service object {} svc, () attachPoint = ()) returns error? {
        var _ = svc;
        var _ = attachPoint;
        io:println("attach ", self.n);
    }

    public function detach(service object {} svc) returns error? {
        var _ = svc;
    }

    public function 'start() returns error? {
        io:println("start ", self.n);
    }

    public function gracefulStop() returns error? {
        io:println("gracefulStop ", self.n);
    }

    public function immediateStop() returns error? {
    }
}

function makeListener(string n) returns L {
    io:println("make ", n);
    return new L(n);
}

// Each on-clause call expression is evaluated when its service is
// initialized, and each resulting object is registered once.
service on makeListener("a") { // @output make a
    // @output listener init a
    // @output attach a
}

service on makeListener("b") { // @output make b
    // @output listener init b
    // @output attach b
}

public function main() {
    io:println("main"); // @output main
    // @output start a
    // @output start b
    // @output gracefulStop a
    // @output gracefulStop b
}
