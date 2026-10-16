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

type Info record {|
    int value;
|};

type Flag record {|
    boolean value;
|};

annotation Info serviceInfo on service;
annotation Flag serviceFlag on service;
annotation Info fieldInfo on object field;
annotation Flag methodFlag on object function;
annotation Info returnInfo on return;

class Listener {
    public function attach(service object {} svc, string[]|string? attachPoint = ()) returns error? {
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

@serviceInfo {value: "str"} // @error
service on new Listener() {
    @fieldInfo {value: true} // @error
    int count = 0;

    @methodFlag {value: 1 == "a"} // @error
    resource function get count() returns @returnInfo {value: "str"} int { // @error
        return self.count;
    }
}

@serviceFlag {value: 1 == "a"} // @error
service on new Listener() {
}

public function main() {
}
