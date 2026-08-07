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

type FirstService distinct service object {};
type SecondService distinct service object {};
type ServiceUnion FirstService|SecondService;

class UnionListener {
    public function attach(ServiceUnion svc, () attachPoint = ()) returns error? {
        var _ = svc;
        var _ = attachPoint;
    }

    public function detach(ServiceUnion svc) returns error? {
        var _ = svc;
    }

    public function 'start() returns error? {}
    public function gracefulStop() returns error? {}
    public function immediateStop() returns error? {}
}

listener UnionListener explicitListener = new;
listener UnionListener inferredListener = new;

service ServiceUnion on explicitListener { // @error explicit service type is not atomic
}

service on inferredListener { // @error inferred service type is not atomic
}

public function main() {
}
