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

class Bar {
    int val = 0;

    function init(int v) returns error? {
        if v < 0 {
            return error("value must be non-negative");
        }
        self.val = v;
    }

    function getVal() returns int {
        return self.val;
    }
}

public function main() {
    Bar|error r1 = new (5);
    if r1 is Bar {
        io:println(r1.getVal()); // @output 5
    }

    Bar|error r2 = new (-1);
    if r2 is error {
        io:println(r2); // @output error("value must be non-negative")
    }
}
