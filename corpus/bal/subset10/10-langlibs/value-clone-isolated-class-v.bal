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
import ballerina/lang.value;

isolated class Store {
    private int[] xs = [];

    isolated function set(int[] v) {
        lock {
            self.xs = value:clone(v);
        }
    }

    isolated function get() returns int[] {
        lock {
            return self.xs.clone();
        }
    }
}

public function main() {
    Store s = new;
    int[] v = [1, 2];
    s.set(v);
    v.push(3);
    int[] got = s.get();
    got.push(4);
    io:println(v); // @output [1,2,3]
    io:println(got); // @output [1,2,4]
    io:println(s.get()); // @output [1,2]
}
