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

class First {
    int a;

    function init(record {| int a; |} value, int marker) {
        self.a = value.a + marker;
    }
}

class Second {
    int a;

    function init(record {| int a; |} value, string marker) {
        self.a = value.a + marker.length();
    }
}

public function main() {
    record {| int a; |} value = {a: 2};
    any y = value;
    First|Second c = new (<record {| int a = 1; |}>y, 1);
    io:println(c is First ? "First" : "Second"); // @output First
    io:println(c.a); // @output 3
}
