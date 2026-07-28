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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

public isolated class IntAttachPointListener {
    public isolated function attach(service object {} svc, int attachPoint = 0) returns error? {
        var _ = svc;
        var _ = attachPoint;
    }

    public isolated function 'start() returns error? {}

    public isolated function gracefulStop() returns error? {}

    public isolated function immediateStop() returns error? {}

    public isolated function detach(service object {} svc) returns error? {
        var _ = svc;
    }
}

public isolated class IntArrayAttachPointListener {
    public isolated function attach(service object {} svc, int[] attachPoint = []) returns error? {
        var _ = svc;
        var _ = attachPoint;
    }

    public isolated function 'start() returns error? {}

    public isolated function gracefulStop() returns error? {}

    public isolated function immediateStop() returns error? {}

    public isolated function detach(service object {} svc) returns error? {
        var _ = svc;
    }
}

service on new IntAttachPointListener() { // @error listener attach-point type cannot be int
}

service /foo on new IntArrayAttachPointListener() { // @error listener attach-point type cannot be int[]
}
