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

final L x = new ("x");
final L y = new ("y");

// Module listeners are started and stopped in the order they were first
// registered; registering an object again does not move it.
service on y { // @output attach y
}

service on x, y { // @output attach x
    // @output attach y
}

public function main() {
    io:println("main"); // @output main
    // @output start y
    // @output start x
    // @output gracefulStop y
    // @output gracefulStop x
}
