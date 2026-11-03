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

type Word "def"|"ghi";

public function main() {
    xml text = xml `abc`;
    string suffix = "def";

    io:println(text + suffix); // @output abcdef
    io:println(suffix + text); // @output defabc
    io:println(text + "<item/>"); // @output abc&lt;item/&gt;
    string ampersand = "&amp;";
    string lessThan = "&lt;";
    io:println(text + ampersand); // @output abc&amp;amp;
    io:println(ampersand + text); // @output &amp;amp;abc
    io:println(text + lessThan); // @output abc&amp;lt;
    io:println(lessThan + text); // @output &amp;lt;abc

    xml:Text textSubtype = xml `text`;
    string:Char charSubtype = "x";
    Word wordSubtype = "ghi";

    xml:Text appendedText = textSubtype + charSubtype;
    xml:Text prependedText = charSubtype + textSubtype;
    xml<xml:Text> textSequence = textSubtype + wordSubtype;
    xml<xml:Text> prependedSequence = wordSubtype + textSubtype;
    io:println(appendedText); // @output textx
    io:println(prependedText); // @output xtext
    io:println(textSequence); // @output textghi
    io:println(prependedSequence); // @output ghitext

    xml:Element element = xml `<item/>`;
    xml<xml:Element|xml:Text> elementWithText = element + suffix;
    xml<xml:Element|xml:Text> textWithElement = suffix + element;
    io:println(elementWithText); // @output <item/>def
    io:println(textWithElement); // @output def<item/>
}
