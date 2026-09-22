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

type Meta record {|
    string name;
|};

annotation Meta serviceMeta on service;
annotation Meta runtimeServiceMeta on service;
annotation Meta parameterMeta on parameter;

function annotationName() returns string {
    return "runtime-service";
}

function alphaName() returns string {
    return "runtime-alpha";
}

function betaName() returns string {
    return "runtime-beta";
}

class SimpleListener {
    public function attach(service object {} svc, string[]|string? attachPoint = ()) returns () {
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

@serviceMeta {name: "declaration"}
@runtimeServiceMeta {name: annotationName()}
service /annotated on new SimpleListener() {
    function alpha(@parameterMeta {name: alphaName()} int value) returns int {
        return value;
    }

    function beta(@parameterMeta {name: betaName()} int value) returns int {
        return value;
    }

    resource function get value() returns int {
        return self.alpha(self.beta(42));
    }
}

public function main() {
    io:println("service annotations retained"); // @output service annotations retained
}
