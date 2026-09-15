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

import ballerina/http;

class CustomListener {
    public function attach(service object {} svc, string[] attachPoint = []) returns error? {
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

service /invalid on new http:Listener(19222) {
    resource function get wrongType(int value) { // @error
        var _ = value;
    }

    resource function post tooMany(http:Request first, http:Request second) { // @error
        var _ = first;
        var _ = second;
    }

    resource function put rest(http:Request... requests) { // @error
        var _ = requests;
    }
}

service /custom on new CustomListener() {
    resource function get ignored(int value) {
        var _ = value;
    }
}

service /mixed on new CustomListener(), new http:Listener(19223) {
    resource function patch checked(string value) { // @error
        var _ = value;
    }
}
