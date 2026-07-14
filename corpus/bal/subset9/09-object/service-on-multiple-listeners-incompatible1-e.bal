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

type FooService service object {
    function foo() returns int;
};

type BarService service object {
    function bar() returns int;
};

class FooListener {
    public function attach(FooService svc, () attachPoint = ()) returns () {
        var _ = svc;
        var _ = attachPoint;
    }

    public function detach(FooService svc) returns error? {
        var _ = svc;
    }

    public function 'start() returns error? {
    }

    public function gracefulStop() returns error? {
    }

    public function immediateStop() returns error? {
    }
}

class BarListener {
    public function attach(BarService svc, () attachPoint = ()) returns () {
        var _ = svc;
        var _ = attachPoint;
    }

    public function detach(BarService svc) returns error? {
        var _ = svc;
    }

    public function 'start() returns error? {
    }

    public function gracefulStop() returns error? {
    }

    public function immediateStop() returns error? {
    }
}

listener FooListener fooListener = new ();
listener BarListener barListener = new ();

service on fooListener, barListener { // @error service must satisfy both listener target service types
    function foo() returns int {
        return 1;
    }
}

public function main() {
}
