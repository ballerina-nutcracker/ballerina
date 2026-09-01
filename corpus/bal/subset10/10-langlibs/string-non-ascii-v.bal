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

// The Ascii functions are deliberately ASCII-only, per their spec wording:
// "Converts occurrences of A-Z to a-z. Other characters are left unchanged."
//
// This is a known, intentional divergence from jBallerina, which implements
// them with Java's locale-sensitive toLowerCase/toUpperCase and so also folds
// non-ASCII (it maps "HÉLLO" to "héllo", grows "ß" to "SS", and under a Turkish
// default locale maps "I" to a dotless "ı"). Likewise its trim() uses Java's
// String.trim(), which strips every code unit <= U+0020 including the C0
// controls, where the spec limits the set to 0x9..0xD and 0x20.
public function main() {
    // Non-ASCII letters are left untouched by case conversion.
    io:println("HÉLLO".toLowerAscii()); // @output hÉllo
    io:println("héllo".toUpperAscii()); // @output HéLLO
    io:println("ß".toUpperAscii()); // @output ß
    io:println("ΣΟΦΟΣ".toLowerAscii()); // @output ΣΟΦΟΣ
    io:println("日本語".toUpperAscii()); // @output 日本語

    // ASCII inside a non-ASCII string still converts.
    io:println("aÉb".toUpperAscii()); // @output AÉB

    // Case-insensitive comparison folds ASCII only, so accented pairs differ.
    io:println("HÉLLO".equalsIgnoreCaseAscii("héllo")); // @output false
    io:println("HELLO".equalsIgnoreCaseAscii("hello")); // @output true

    // trim removes only the ASCII whitespace set, not the C0 controls.
    string ctl = "\u{0001}abc\u{001F}";
    io:println(ctl.length()); // @output 5
    io:println(ctl.trim().length()); // @output 5
    io:println("\u{0009}\u{000A} abc \u{000D}".trim()); // @output abc

    // Code points, not UTF-16 units: an astral character counts as one.
    io:println("a😀b".length()); // @output 3
    io:println("a😀b".substring(1, 2)); // @output 😀
    io:println("héllo😀".toBytes().length()); // @output 10
}
