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

// The XML subtypes (Element, Comment, Text, ProcessingInstruction) are
// built-in types that cannot be expressed in source; they are provided through
// the compiler's opaque-symbol mechanism.

public const string XML_NAMESPACE_URI = "http://www.w3.org/XML/1998/namespace";
public const string XMLNS_NAMESPACE_URI = "http://www.w3.org/2000/xmlns/";
public const string space = "{http://www.w3.org/XML/1998/namespace}space";
public const string lang = "{http://www.w3.org/XML/1998/namespace}lang";
public const string base = "{http://www.w3.org/XML/1998/namespace}base";

public isolated function length(xml x) returns int = external;
public isolated function concat((xml|string)... xs) returns xml = external;
public isolated function getName(Element elem) returns string = external;
public isolated function setName(Element elem, string xName) = external;
public isolated function getAttributes(Element x) returns map<string> = external;
public isolated function getChildren(Element elem) returns xml = external;
public isolated function setChildren(Element elem, xml|string children) = external;
public isolated function getDescendants(Element elem) returns xml = external;
public isolated function data(xml x) returns string = external;
public isolated function getTarget(ProcessingInstruction x) returns string = external;
public isolated function getContent(ProcessingInstruction|Comment x) returns string = external;

public isolated function createElement(string name,
        map<string> attributes = {}, xml children = xml``) returns Element = external;
public isolated function createProcessingInstruction(string target, string content)
        returns ProcessingInstruction = external;
public isolated function createComment(string content) returns Comment = external;
public isolated function createText(string data) returns Text = external;

public isolated function strip(xml x) returns xml = external;
public isolated function elements(xml x, string? nm = ()) returns xml<Element> = external;
public isolated function children(xml x) returns xml = external;
public isolated function elementChildren(xml x, string? nm = ()) returns xml<Element> = external;
public isolated function text(xml x) returns Text = external;
public isolated function fromString(string s) returns xml|error = external;
