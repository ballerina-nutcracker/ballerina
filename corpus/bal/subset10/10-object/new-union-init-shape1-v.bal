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

class ByInt {
    string tag;

    function init(int n) {
        self.tag = n.toString();
    }

    function name() returns string {
        return "ByInt:" + self.tag;
    }
}

class ByString {
    string tag;

    function init(string s) {
        self.tag = s;
    }

    function name() returns string {
        return "ByString:" + self.tag;
    }
}

public function main() {
    ByInt|ByString a = new (5);
    io:println(a.name()); // @output ByInt:5
    ByInt|ByString b = new ("s");
    io:println(b.name()); // @output ByString:s
    ByInt c = new ByString("t");
    io:println(c.name()); // @output ByString:t
}
