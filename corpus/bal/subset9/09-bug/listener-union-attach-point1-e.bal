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

type S service object {};

class AmbiguousListener {
    public function attach(S svc, [string] | string[] attachPoint) returns error? {
        _ = svc;
        _ = attachPoint;
    }

    public function detach(S svc) returns error? {
        _ = svc;
    }

    public function 'start() returns error? {}

    public function gracefulStop() returns error? {}

    public function immediateStop() returns error? {}
}

class UnsupportedListener {
    public function attach(S svc, [string] | [string, string] attachPoint) returns error? {
        _ = svc;
        _ = attachPoint;
    }

    public function detach(S svc) returns error? {
        _ = svc;
    }

    public function 'start() returns error? {}

    public function gracefulStop() returns error? {}

    public function immediateStop() returns error? {}
}

listener AmbiguousListener ambiguousListener = new;
listener UnsupportedListener unsupportedListener = new;

service S /foo on ambiguousListener { // @error multiple applicable attach-point list types
}

service S /foo/bar/baz on unsupportedListener { // @error unsupported attach point
}
