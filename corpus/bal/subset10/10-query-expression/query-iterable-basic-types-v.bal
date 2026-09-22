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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/io;

public function main() {
    string[] chars = from string char in "A\u{1F600}\u{0D85}"
        select char;
    io:println(chars.length()); // @output 3
    io:println(chars[0] == "A"); // @output true
    io:println(chars[1] == "\u{1F600}"); // @output true
    io:println(chars[2] == "\u{0D85}"); // @output true

    string[] ordered = from string char in "cba"
        where char != "b"
        order by char
        limit 2
        select char;
    io:println(ordered); // @output ["a","c"]

    xml<xml:Element|xml:Text|xml:Comment> items = xml `<a/>text<!--note-->`;
    xml[] selectedItems = from var item in items
        select item;
    foreach xml item in selectedItems {
        io:println(item); // @output <a/>
                          // @output text
                          // @output <!--note-->
    }

    [string, xml:Element][] products = from string char in "xy"
        from xml:Element element in xml `<a/><b/>`
        select [char, element];
    foreach [string, xml:Element] product in products {
        io:println(product[0], product[1]); // @output x<a/>
                                                  // @output x<b/>
                                                  // @output y<a/>
                                                  // @output y<b/>
    }

    string[] matchedChars = from string expected in "bd"
        join string actual in "abcd"
        on expected equals actual
        select actual;
    io:println(matchedChars); // @output ["b","d"]

    xml<xml:Element> expectedElements = xml `<b/>`;
    xml:Element[] matchedElements = from xml:Element left in expectedElements
        join xml:Element actual in xml `<a/><b/>`
        on left equals actual
        select actual;
    io:println(matchedElements[0]); // @output <b/>
}
