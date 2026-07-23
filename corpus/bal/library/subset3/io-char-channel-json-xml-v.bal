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
    // JSON roundtrip
    string jpath = "/tmp/bal_io_char_json.json";
    io:WritableByteChannel wjb = check io:openWritableFile(jpath);
    io:WritableCharacterChannel wjc = new (wjb, "UTF-8");
    check wjc.writeJson({name: "Ballerina", age: 12});
    check wjc.close();

    io:ReadableByteChannel rjb = check io:openReadableFile(jpath);
    io:ReadableCharacterChannel rjc = new (rjb, "UTF-8");
    json j = check rjc.readJson();
    map<json> jm = <map<json>>j;
    io:println(jm["name"]); // @output Ballerina
    io:println(jm["age"]); // @output 12
    check rjc.close();

    // malformed JSON errors
    string bpath = "/tmp/bal_io_char_json_bad.json";
    check io:fileWriteString(bpath, "{oops");
    io:ReadableByteChannel bjb = check io:openReadableFile(bpath);
    io:ReadableCharacterChannel bjc = new (bjb, "UTF-8");
    json|io:Error bad = bjc.readJson();
    io:println(bad is io:Error); // @output true
    check bjc.close();

    // XML with each doctype variant
    xml note = xml `<note><to>Tove</to></note>`;
    string xpath = "/tmp/bal_io_char_xml.xml";

    io:WritableByteChannel wxb1 = check io:openWritableFile(xpath);
    io:WritableCharacterChannel wxc1 = new (wxb1, "UTF-8");
    check wxc1.writeXml(note, {system: "note.dtd"});
    check wxc1.close();
    io:println(check io:fileReadString(xpath)); // @output <!DOCTYPE note SYSTEM "note.dtd">
    // @output <note><to>Tove</to></note>

    io:WritableByteChannel wxb2 = check io:openWritableFile(xpath);
    io:WritableCharacterChannel wxc2 = new (wxb2, "UTF-8");
    check wxc2.writeXml(note, {'public: "-//W3C//DTD//EN", system: "note.dtd"});
    check wxc2.close();
    io:println(check io:fileReadString(xpath)); // @output <!DOCTYPE note PUBLIC "-//W3C//DTD//EN" "note.dtd">
    // @output <note><to>Tove</to></note>

    io:WritableByteChannel wxb3 = check io:openWritableFile(xpath);
    io:WritableCharacterChannel wxc3 = new (wxb3, "UTF-8");
    check wxc3.writeXml(note, {internalSubset: "[<!ELEMENT to (#PCDATA)>]"});
    check wxc3.close();
    io:println(check io:fileReadString(xpath)); // @output <!DOCTYPE note [<!ELEMENT to (#PCDATA)>]>
    // @output <note><to>Tove</to></note>

    // no doctype
    io:WritableByteChannel wxb4 = check io:openWritableFile(xpath);
    io:WritableCharacterChannel wxc4 = new (wxb4, "UTF-8");
    check wxc4.writeXml(note);
    check wxc4.close();
    io:println(check io:fileReadString(xpath)); // @output <note><to>Tove</to></note>

    // readXml parses the element back
    io:ReadableByteChannel rxb = check io:openReadableFile(xpath);
    io:ReadableCharacterChannel rxc = new (rxb, "UTF-8");
    xml parsed = check rxc.readXml();
    io:println(parsed); // @output <note><to>Tove</to></note>
    check rxc.close();
}
