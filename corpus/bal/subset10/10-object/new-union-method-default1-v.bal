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

type Run object {
    function run(int p) returns int;
};

class Runner {
    function run(int p = 0) returns int {
        return p + 1;
    }
}

class First {
    int result;

    function init(Run runner, int marker) {
        self.result = runner.run(marker);
    }
}

class Second {
    int result;

    function init(Run runner, string marker) {
        self.result = runner.run(marker.length());
    }
}

public function main() {
    any v = new Runner();
    First|Second a = new (<object { function run(int p = 3) returns int; }>v, 4);
    First|Second b = new (<object { function run(int p = 3) returns int; }>v, "ab");
    io:println(a is First ? "First" : "Second"); // @output First
    io:println(a.result); // @output 5
    io:println(b is First ? "First" : "Second"); // @output Second
    io:println(b.result); // @output 3
}
