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

final L fx = new ("fx");
L vx = new ("vx");

// A listener held in a non-listener module variable is registered by the
// service that is attached to it, once however often it is attached.
service on fx, fx { // @output attach fx
    // @output attach fx
}

service on fx, vx { // @output attach fx
    // @output attach vx
}

public function main() {
    io:println("main"); // @output main
    // @output start fx
    // @output start vx
    // @output gracefulStop fx
    // @output gracefulStop vx
}
