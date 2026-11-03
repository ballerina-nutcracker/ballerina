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
import ballerina/mime;

public function main() returns error? {
    mime:Entity latin1Text = new;
    latin1Text.setText("hé", "text/plain; charset=iso-8859-1");
    io:println(check latin1Text.getByteArray()); // @output [104,233]

    mime:Entity latin1Xml = new;
    latin1Xml.setXml(xml `<a>é</a>`, "application/xml; charset=iso-8859-1");
    io:println(check latin1Xml.getByteArray()); // @output [60,97,62,233,60,47,97,62]

    mime:Entity latin1Json = new;
    latin1Json.setJson({"a": "é"}, "application/json; charset=iso-8859-1");
    io:println(check latin1Json.getByteArray()); // @output [123,34,97,34,58,34,233,34,125]

    mime:Entity utf16 = new;
    utf16.setText("hé", "text/plain; charset=utf-16");
    io:println(check utf16.getByteArray()); // @output [254,255,0,104,0,233]

    mime:Entity unmappable = new;
    unmappable.setText("h€", "text/plain; charset=iso-8859-1");
    io:println(check unmappable.getByteArray()); // @output [104,63]

    mime:Entity unknown = new;
    unknown.setText("hé", "text/plain; charset=bogus");
    byte[]|mime:ParserError r = unknown.getByteArray();
    io:println(r is mime:ParserError ? r.message() : r); // @output Error occurred while extracting blob data from entity: bogus

    mime:Entity rawBytes = new;
    rawBytes.setByteArray([104, 233], "text/plain; charset=iso-8859-1");
    io:println((check rawBytes.getByteArray())); // @output [104,233]
    io:println((check rawBytes.getText()).toBytes()); // @output [104,239,191,189]

    mime:Entity malformed = new;
    malformed.setByteArray([60, 97, 62, 233, 233, 65, 226, 130, 66, 60, 47, 97, 62], "application/xml");
    io:println((check malformed.getText()).toBytes()); // @output [60,97,62,239,191,189,239,191,189,65,239,191,189,66,60,47,97,62]
    io:println(check malformed.getXml()); // @output <a>��A�B</a>

    mime:Entity controls = new;
    controls.setJson({"a": "x\u{0008}y\u{000C}"});
    io:println(check controls.getText()); // @output {"a":"x\by\f"}
}
