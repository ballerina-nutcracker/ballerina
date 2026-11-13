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
        io:println("listener init ", n); // @output listener init one
    }

    public function attach(service object {} svc, () attachPoint = ()) returns error? {
        var _ = svc;
        var _ = attachPoint;
        io:println("attach ", self.n); // @output attach one
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

function makeListener(string n) returns L|error {
    io:println("make ", n); // @output make two
    return error("make failed " + n);
}

service on new L("one"), makeListener("two") { // @panic make failed two
}

public function main() {
}
