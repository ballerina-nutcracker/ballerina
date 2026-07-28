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

import ballerina/io;

public type Service service object {};

public isolated class Listener {
    public isolated function attach(Service svc, string[]|string? name = ()) returns error? {
        var _ = svc;
        if name is string {
            io:println(name); // @output watch
        } else if name is string[] {
            io:println(name[0]); // @output foo
            io:println(name[1]); // @output bar
        } else {
            io:println("root"); // @output root
        }
    }

    public isolated function 'start() returns error? {}

    public isolated function gracefulStop() returns error? {}

    public isolated function immediateStop() returns error? {}

    public isolated function detach(Service svc) returns error? {
        var _ = svc;
    }
}

service "watch" on new Listener() {
}

service /foo/bar on new Listener() {
}

service on new Listener() {
}

public function main() {
}
