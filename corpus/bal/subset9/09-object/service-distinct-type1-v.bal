// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/io;

public type ServiceType distinct service object {
    public function value() returns int;
};

class Listener {
    public function attach(ServiceType svc, string[] attachPoint) returns error? {
        service object {} broad = svc;
        io:println(broad is ServiceType); // @output true
        io:println(svc.value()); // @output 41
        io:println(attachPoint.length()); // @output 0
    }

    public function detach(ServiceType svc) returns error? {
        var _ = svc;
    }

    public function 'start() returns error? {
    }

    public function gracefulStop() returns error? {
    }

    public function immediateStop() returns error? {
    }
}

listener Listener l = new;

service / on l {
    private int base = 40;

    private function increment() returns int {
        return self.base + 1;
    }

    public function value() returns int {
        return self.increment();
    }
}

class ExplicitListener {
    public function attach(ServiceType svc, string[] attachPoint) returns error? {
        service object {} broad = svc;
        io:println(broad is ServiceType); // @output true
        io:println(svc.value()); // @output 42
        io:println(attachPoint.length()); // @output 0
    }

    public function detach(ServiceType svc) returns error? {
        var _ = svc;
    }

    public function 'start() returns error? {
    }

    public function gracefulStop() returns error? {
    }

    public function immediateStop() returns error? {
    }
}

listener ExplicitListener explicitListener = new;

service ServiceType / on explicitListener {
    private int base = 41;

    private function increment() returns int {
        return self.base + 1;
    }

    public function value() returns int {
        return self.increment();
    }
}

public function main() {
}
