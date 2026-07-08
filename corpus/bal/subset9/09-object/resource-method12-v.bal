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

client class Base {
    int tag = 1;

    resource function get items/[string id]() {
        io:println("Base get " + id);
    }
}

// Derived includes Base. Class inclusion brings over fields only, never
// methods; Derived must not be required to redeclare Base's resource
// method, and the absence of that method here must not be an error.
client class Derived {
    *Base;

    function init() {
        self.tag = 2;
    }
}

public function main() {
    Derived d = new ();
    io:println(d.tag); // @output 2
}
