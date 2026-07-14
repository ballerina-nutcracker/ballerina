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

// @productions string-literal equality equality-expr local-var-decl-stmt
import ballerina/io;

public function main() {
    string name = "เจมส์";
    io:println(name); // @output เจมส์
    io:println(name.length()); // @output 5
    string name2 = "\u{0e40}\u{E08}\u{0000e21}\u{0e2a}\u{e4c}";
    io:println(name2); // @output เจมส์
    io:println(name == name2); // @output true
    io:println(name != name2); // @output false
    string name3 = "James";
    _ = name3;
    io:println(name == "James"); // @output false
    io:println(name != "James"); // @output true
}

