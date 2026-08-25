// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

public function main() returns error? {
    // XML with default and prefixed namespace declarations plus a nested
    // qualified element -> exercises nsCtx.child / qualifiedName / xmlns attrs.
    string nsPath = "/tmp/bal_io_xml_ns.xml";
    string nsContent = "<root xmlns=\"http://default\" xmlns:a=\"http://a.com\">" +
        "<a:item a:key=\"v\">text</a:item><child/></root>";
    check io:fileWriteString(nsPath, nsContent);
    xml nsResult = check io:fileReadXml(nsPath);
    io:println(nsResult); // @output <root xmlns="http://default" xmlns:a="http://a.com"><a:item a:key="v">text</a:item><child/></root>

    // Multiple top-level items (comment, processing instruction, two elements)
    // preceded by a DOCTYPE directive -> parseXMLFromBytes default case + directive skip.
    string multiPath = "/tmp/bal_io_xml_multi.xml";
    string multiContent = "<!DOCTYPE note><!-- a comment --><?target data?><a/><b/>";
    check io:fileWriteString(multiPath, multiContent);
    xml multiResult = check io:fileReadXml(multiPath);
    io:println(multiResult); // @output <!-- a comment --><?target data?><a/><b/>

    // Whitespace-only / empty top-level content -> parseXMLFromBytes empty case.
    string emptyPath = "/tmp/bal_io_xml_empty.xml";
    check io:fileWriteString(emptyPath, "   \n  ");
    xml emptyResult = check io:fileReadXml(emptyPath);
    io:println(emptyResult === xml ``); // @output true

    // Malformed XML with an unclosed element -> parse error inside element.
    string badPath = "/tmp/bal_io_xml_bad.xml";
    check io:fileWriteString(badPath, "<root><unclosed></root>");
    xml|io:Error badResult = io:fileReadXml(badPath);
    io:println(badResult is io:Error); // @output true

    // Stray end element at the top level -> unexpected end element error.
    string strayPath = "/tmp/bal_io_xml_stray.xml";
    check io:fileWriteString(strayPath, "</root>");
    xml|io:Error strayResult = io:fileReadXml(strayPath);
    io:println(strayResult is io:Error); // @output true
}
