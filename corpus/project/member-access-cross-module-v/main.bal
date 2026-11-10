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
import testorg/memberaccesscrossmodulev.foo;

class Local {
    *foo:T;
    public int pub = 1;

    public function get() returns int {
        return self.pub + 1;
    }
}

public function main() {
    foo:C c = new;
    c.y += 1;
    io:println(c.y); // @output 3
    io:println(c.get()); // @output 7
    io:println(c.peek(c)); // @output 6
    io:println(foo:readMod(c)); // @output 4
    foo:Cl cl = new;
    int r = cl->r();
    io:println(r); // @output 9
    foo:T t = new Local();
    io:println(t.pub + t.get()); // @output 3
}
