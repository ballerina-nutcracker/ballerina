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

class Plain {
    int x = 1;

    function get() returns int {
        return self.x;
    }
}

class WithArgs {
    int x;

    function init(int x, string label = "l") {
        self.x = x + label.length() - 1;
    }

    function get() returns int {
        return self.x * 10;
    }
}

class Fallible {
    int x;

    function init() returns error? {
        self.x = 3;
    }

    function get() returns int {
        return self.x * 100;
    }
}

function total(Plain p) returns int {
    return p.get();
}

public function main() returns error? {
    Plain p = new WithArgs(2);
    io:println(p.get()); // @output 20
    WithArgs w = new Plain();
    io:println(w.get()); // @output 1
    Fallible f = check new Fallible();
    io:println(total(f)); // @output 300
}
