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
import ballerina/mime;

public function main() returns error? {
    // setXml/getXml round trip; default content-type is application/xml
    mime:Entity e1 = new;
    e1.setXml(xml `<hello>world</hello>`);
    io:println(e1.getContentType()); // @output application/xml
    var r1 = e1.getXml();
    if r1 is xml {
        io:println(r1.toString()); // @output <hello>world</hello>
    }

    // getByteArray() also materializes an XML body, not just getXml()/getText().
    byte[]|mime:ParserError r1Bytes = e1.getByteArray();
    if r1Bytes is byte[] {
        io:println(check string:fromBytes(r1Bytes)); // @output <hello>world</hello>
    }

    // setBody's xml arm dispatches to setXml
    mime:Entity e2 = new;
    e2.setBody(xml `<a><b/></a>`);
    io:println(e2.getContentType()); // @output application/xml

    // getXml() cross-converts a text body that happens to contain XML markup
    mime:Entity e3 = new;
    e3.setText("<x>y</x>");
    var r3 = e3.getXml();
    io:println(r3 is xml); // @output true
    if r3 is xml {
        io:println(r3.toString()); // @output <x>y</x>
    }

    // Malformed markup (unclosed element) is rejected
    mime:Entity e4 = new;
    e4.setText("<root><unclosed></root>");
    var r4 = e4.getXml();
    io:println(r4 is mime:ParserError); // @output true

    // A freshly-constructed entity with no body at all still errors, like the
    // other accessors (getText/getJson/getByteArray).
    mime:Entity e5 = new;
    var r5 = e5.getXml();
    io:println(r5 is mime:ParserError); // @output true

    // Explicit content-type override
    mime:Entity e6 = new;
    e6.setXml(xml `<v/>`, "application/xhtml+xml");
    io:println(e6.getContentType()); // @output application/xhtml+xml
}
