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

class IntFn {
    int result;

    function init(function () returns int callback) {
        self.result = callback();
    }
}

class StringFn {
    string result;

    function init(function () returns string callback) {
        self.result = callback();
    }
}

IntFn|StringFn moduleLevel = new (function () returns int {
    worker w returns int {
        return 2;
    }
    int v = wait w;
    return v + 1;
});

public function main() {
    IntFn|StringFn local = new (function () returns string {
        worker w returns string {
            return "a";
        }
        string v = wait w;
        return v + "b";
    });
    IntFn|StringFn m = moduleLevel;
    if m is IntFn {
        io:println(m.result); // @output 3
    }
    if local is StringFn {
        io:println(local.result); // @output ab
    }
}
