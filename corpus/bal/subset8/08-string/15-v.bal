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

// @productions string string-literal equality equality-expr additive-expr function-call-expr local-var-decl-stmt
import ballerina/io;

public function main() {
    concatTest("", "", ""); // @output 
    // @output 0
    // @output true

    concatTest("a", "b", "ab"); // @output ab
    // @output 2
    // @output true

    concatTest("smile", "\u{1F642}", "smile\u{1F642}"); // @output smile🙂
    // @output 6
    // @output true

    concatTest("\u{1F642}", "frown", "\u{1F642}frown"); // @output 🙂frown
    // @output 6
    // @output true

    concatTest("\u{1F641}", "\u{1F642}", "\u{1F641}\u{1F642}"); // @output 🙁🙂
    // @output 2
    // @output true
}

function concatTest(string s1, string s2, string expected) {
    string s = s1 + s2;
    io:println(s);
    io:println(s.length());
    io:println(s == expected);
}
