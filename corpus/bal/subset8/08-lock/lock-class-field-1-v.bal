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

isolated class Counter {
    private int count;

    function init(int initial) {
        self.count = initial;
    }

    isolated function inc() {
        lock {
            self.count = self.count + 1;
        }
    }

    isolated function get() returns int {
        lock {
            return self.count;
        }
    }
}

public function main() {
    Counter c = new(0);
    c.inc();
    c.inc();
    c.inc();
    io:println(c.get()); // @output 3
}
