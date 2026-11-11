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

public class L {
    private final string name;

    public function init(string name) {
        self.name = name;
    }

    public function attach(service object {} s, () attachPoint = ()) returns error? {
        var _ = s;
        var _ = attachPoint;
        io:println("attach ", self.name);
    }

    public function detach(service object {} s) returns error? {
        var _ = s;
    }

    public function 'start() returns error? {
        io:println("start ", self.name);
    }

    public function gracefulStop() returns error? {
        io:println("gracefulStop ", self.name);
    }

    public function immediateStop() returns error? {
    }
}

// Registered with foo by its listener declaration and again with the root
// module through `fx`.
public listener L fl = new ("fl");
public final L fx = fl;

// Not a module listener of foo; registered with bar and with the root module.
public final L gx = new ("gx");
