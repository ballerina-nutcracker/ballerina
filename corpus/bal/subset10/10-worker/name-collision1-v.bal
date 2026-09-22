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

// Worker closures become package-level generated functions, so their names must
// be unique package-wide, not just within the scope that declares them. These
// declarations all name a worker `w` from an owner whose own name repeats.

class A {
    int value;

    function init() {
        worker w returns int {
            return 1;
        }
        self.value = wait w;
    }
}

class B {
    int value;

    function init() {
        worker w returns int {
            return 2;
        }
        self.value = wait w;
    }
}

function shadowed() returns int {
    worker w returns int {
        return 3;
    }
    return wait w;
}

public function main() {
    A a = new;
    B b = new;
    io:println(a.value); // @output 1
    io:println(b.value); // @output 2
    io:println(shadowed()); // @output 3
}
