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

type IntValueService service object {
    int value;
};

type StringValueService service object {
    string value;
};

class IntValueListener {
    public function attach(IntValueService svc, () attachPoint = ()) returns error? {
        var _ = svc;
        var _ = attachPoint;
    }

    public function detach(IntValueService svc) returns error? {
        var _ = svc;
    }

    public function 'start() returns error? {}
    public function gracefulStop() returns error? {}
    public function immediateStop() returns error? {}
}

class StringValueListener {
    public function attach(StringValueService svc, () attachPoint = ()) returns error? {
        var _ = svc;
        var _ = attachPoint;
    }

    public function detach(StringValueService svc) returns error? {
        var _ = svc;
    }

    public function 'start() returns error? {}
    public function gracefulStop() returns error? {}
    public function immediateStop() returns error? {}
}

listener IntValueListener intValueListener = new;
listener StringValueListener stringValueListener = new;

service on intValueListener, stringValueListener { // @error listener service types have an empty intersection
}

service on intValueListener {
    int value = 0;

    resource function get items/[any path]() { // @error resource path parameter is not anydata
    }
}

service on intValueListener {
    int value = 0;
    never impossible; // @error service definition is empty
}

public function main() {
}
