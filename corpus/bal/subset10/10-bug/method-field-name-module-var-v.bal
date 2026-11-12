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

int base = 7;

class P {
    int base = 3;

    function sum(int a, int b = base) returns int => a + b;

    function get() returns int => base;

    function getField() returns int => self.base;
}

public function main() {
    P p = new;
    io:println(p.sum(1)); // @output 8
    io:println(p.get()); // @output 7
    io:println(p.getField()); // @output 3
}
