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
    string s = "location.buildingCode";

    io:println(s.includes(".")); // @output true
    io:println(s.indexOf(".")); // @output 8
    io:println(s.includes("[]")); // @output false
    io:println(s.indexOf("[]")); // @output

    // startIndex skips past the first match
    string multi = "a.b.c";
    io:println(multi.indexOf(".")); // @output 1
    io:println(multi.indexOf(".", 2)); // @output 3
    io:println(multi.includes(".", 4)); // @output false

    // multi-byte codepoints: indices count runes, not bytes
    string unicode = "héllo wörld";
    io:println(unicode.indexOf("wörld")); // @output 6
    io:println(unicode.includes("ö")); // @output true

    // boundary cases
    io:println("".includes("")); // @output true
    io:println("".indexOf("")); // @output 0
    io:println("abc".indexOf("", 5)); // @output 3
    io:println("abc".indexOf("x", -5)); // @output
}
