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

// A listener object registered with several modules is a single module
// listener, so it is started and stopped once (spec 8.3).

// @output attach gx
// @output attach fl
// @output attach gx
// @output bar
// @output main
// @output start fl
// @output start gx
// @output gracefulStop gx
// @output gracefulStop fl

import ballerina/io;

import testorg/xmodlistenerregisteredonce.bar;
import testorg/xmodlistenerregisteredonce.foo;

service on foo:fx {
}

service on foo:gx {
}

public function main() {
    bar:hello();
    io:println("main");
}
