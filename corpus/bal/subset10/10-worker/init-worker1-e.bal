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

// Spec 2024R1 §6.8.1.2 waives the isolated-object lock requirement inside
// `init`, but only at points where some field is potentially uninitialized. The
// waiver covers the default worker's own statements at such a point; it must not
// reach into a named worker body, which runs on its own strand and can outlive
// init, so the object it mutates is reachable from another strand.
isolated class Holder {
    private int[] arr;
    private int n;

    function init() {
        self.arr = [];
        // Legal: `n` is still potentially uninitialized here, so the waiver is
        // live and this unguarded access needs no lock.
        self.arr.push(1);
        worker w {
            self.arr.push(2); // @error the waiver does not reach a worker body
        }
        _ = wait w;
        self.n = 0;
    }

    isolated function count() returns int {
        lock {
            return self.arr.length();
        }
    }
}

public function main() {
    Holder h = new;
    io:println(h.count());
}
