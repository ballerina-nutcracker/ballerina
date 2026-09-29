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

type T object {
    int x;
};

class C {
    int x = 1;
    private int y = 2;

    function sum() returns int {
        return self.x + self.y;
    }
}

public function main() {
    any a = new C();
    if a !is T {
        io:println("not T");
    } else if a is C {
        C c = a;
        io:println(c.sum()); // @output 3
    }
    T|int v = new C();
    if v is int {
        io:println(v);
    } else {
        io:println(v.x); // @output 1
    }
}
