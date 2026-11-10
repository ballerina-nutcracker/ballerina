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

class SimpleListener {
    public function attach(service object {} svc, () attachPoint = ()) returns () {
        var _ = svc;
        var _ = attachPoint;
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

class A {
    private int x = 1;
}

service on new SimpleListener() {
    private int count = 0;

    private function helper() returns int {
        return self.count;
    }

    remote function update(A a) {
        self.count += self.helper();
        self.count += a.x; // @error private field of A is not visible in a service method
    }

    resource function get value(A a) returns int {
        return a.x; // @error private field of A is not visible in a service resource method
    }
}

public function main() {
}
