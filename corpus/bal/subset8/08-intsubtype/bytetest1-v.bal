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

public function main() {
    b(0); // @output true
    b(1); // @output true
    b(254); // @output true
    b(255); // @output true
    b(128); // @output true
    b(127); // @output true
    b(-1); // @output false
    b(256); // @output false
    b(257); // @output false
    b(-2); // @output false
    b(0x10000); // @output false
    b(0x100000000); // @output false
}

function b(int n) {
    io:println(n is byte);
}
