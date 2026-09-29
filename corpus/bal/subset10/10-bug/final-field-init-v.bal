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

class Counter {
    final int a;
    final int b;
    final int[] values = [0];
    int count = 0;

    function init(int a) {
        self.a = a;
        if a > 0 {
            self.b = 1;
        } else {
            self.b = 2;
        }
    }

    function add(int v) {
        self.values[0] = v;
        self.values.push(v);
        self.count += 1;
    }
}

public function main() {
    Counter c = new (5);
    c.add(7);
    io:println(c.a); // @output 5
    io:println(c.b); // @output 1
    io:println(c.values); // @output [7,7]
    io:println(c.count); // @output 1
}
