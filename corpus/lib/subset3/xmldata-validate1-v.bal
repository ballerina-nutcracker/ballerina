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

import ballerina/data.xmldata;
import ballerina/io;

@xmldata:Name {value: "item"}
type Item record {|
    @xmldata:Attribute
    string code;
    @xmldata:Name {value: "item-name"}
    string itemName;
    int count?;
|};

type Open record {
    string id;
};

public function main() {
    io:println(xmldata:validate(xml `<item code="A1"><item-name>Pen</item-name></item>`, Item));
    // @output
    io:println(xmldata:validate(xml `<item code="A1"><item-name>Pen</item-name><count>2</count></item>`, Item));
    // @output
    xmldata:Error? missingAttribute = xmldata:validate(xml `<item><item-name>Pen</item-name></item>`, Item);
    io:println(missingAttribute is xmldata:Error ? missingAttribute.message() : "valid");
    // @output Invalid XML found: 'required attribute 'code' not present in XML'
    xmldata:Error? wrongName = xmldata:validate(xml `<thing code="A1"><item-name>Pen</item-name></thing>`, Item);
    io:println(wrongName is xmldata:Error ? wrongName.message() : "valid");
    // @output Invalid XML found: 'record type name 'item' mismatch with given XML name 'thing''
    xmldata:Error? badCount = xmldata:validate(xml `<item code="A1"><item-name>Pen</item-name><count>x</count></item>`, Item);
    io:println(badCount is xmldata:Error ? badCount.message() : "valid");
    // @output Invalid XML found: ''string' value 'x' cannot be converted to 'int''
    io:println(xmldata:validate(xml `<r><id>1</id><extra>e</extra></r>`, Open));
    // @output
    xmldata:Error? noRoot = xmldata:validate(xml `text`, Open);
    io:println(noRoot is xmldata:Error ? noRoot.message() : "valid");
    // @output Invalid XML found: 'xml root element is missing'
}
