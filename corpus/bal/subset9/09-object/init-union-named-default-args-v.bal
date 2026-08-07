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
    private string label = "";
    private int count = 0;

    function init(string label = "default", int count = 5) returns error? {
        if count < 0 {
            return error("negative count");
        }
        self.label = label;
        self.count = count;
    }

    function describe() returns string {
        return string `${self.label}:${self.count}`;
    }
}

public function main() {
    Counter|error defaulted = new ();
    if defaulted is Counter {
        io:println(defaulted.describe()); // @output default:5
    }

    Counter|error named = new (label = "named", count = 7);
    if named is Counter {
        io:println(named.describe()); // @output named:7
    }

    Counter|error failed = new (count = -1);
    io:println(failed is error); // @output true
}
