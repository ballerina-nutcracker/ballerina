// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import ballerina/io;

int callbackCount = 0;
int filterFailureCount = 0;
int forEachFailureCount = 0;

function failSecond(xml:Element|xml:Text item) returns xml {
    callbackCount += 1;
    if callbackCount == 2 {
        panic error("stop");
    }
    return item;
}

function failFilterSecond(xml:Element|xml:Text item) returns boolean {
    _ = item;
    filterFailureCount += 1;
    if filterFailureCount == 2 {
        panic error("stop filter");
    }
    return true;
}

function failForEachSecond(xml:Element|xml:Text item) {
    _ = item;
    forEachFailureCount += 1;
    if forEachFailureCount == 2 {
        panic error("stop forEach");
    }
}

public function main() {
    string[] invalid = ["<a>", "<?xml version=\"1.0\"?><a/>", "<!DOCTYPE a><a/>",
        "<?XmL bad?>", "<p:a/>", "<a xmlns:p=\"\"/>", "<a xmlns:xml=\"urn:bad\"/>"];
    foreach string xmlSource in invalid {
        xml|error result = xml:fromString(xmlSource);
        io:println(result is error); // @output true
                                      // @output true
                                      // @output true
                                      // @output true
                                      // @output true
                                      // @output true
                                      // @output true
    }

    xml parsed = checkpanic xml:fromString("<outer xmlns:p=\"urn:u\" xmlns:q=\"urn:u\"><p:child q:a=\"v\"/></outer>");
    xml parsedEmpty = checkpanic xml:fromString("");
    io:println(parsedEmpty.length()); // @output 0
    xml:Element child = <xml:Element>parsed.children().get(0);
    io:println(child); // @output <p:child p:a="v" xmlns:p="urn:u" xmlns:q="urn:u"/>
    io:println(child.getAttributes()); // @output {"{urn:u}a":"v","{http://www.w3.org/2000/xmlns/}p":"urn:u","{http://www.w3.org/2000/xmlns/}q":"urn:u"}

    xml a = checkpanic xml:fromString("<p:x xmlns:p=\"urn:u\" xmlns:q=\"urn:u\"/>");
    xml b = checkpanic xml:fromString("<q:x xmlns:p=\"urn:u\" xmlns:q=\"urn:u\"/>");
    xml c = checkpanic xml:fromString("<p:x xmlns:p=\"urn:v\" xmlns:q=\"urn:v\"/>");
    io:println(a == b); // @output true
    io:println(a == c); // @output false

    xml:Element element = xml:createElement("original", {}, xml``);
    error? nameFailure = trap element.setName("{bad");
    io:println(nameFailure is error); // @output true
    io:println(element.getName()); // @output original

    map<string> badAttrs = {"xmlns": "urn:bad"};
    xml emptyChildren = xml``;
    error|xml:Element constructorFailure = trap xml:createElement("x", badAttrs, emptyChildren);
    io:println(constructorFailure is error); // @output true
    io:println(badAttrs); // @output {"xmlns":"urn:bad"}

    error|xml:Comment commentFailure = trap xml:createComment("bad--comment");
    error|xml:ProcessingInstruction piFailure = trap xml:createProcessingInstruction("XML", "bad");
    io:println(commentFailure is error); // @output true
    io:println(piFailure is error); // @output true

    xml<xml:Element|xml:Text> callbackSource = xml `<a/>text<b/>`;
    error|xml callbackFailure = trap callbackSource.map(failSecond);
    io:println(callbackFailure is error); // @output true
    io:println(callbackCount); // @output 2
    error|xml filterFailure = trap callbackSource.filter(failFilterSecond);
    error? forEachFailure = trap callbackSource.forEach(failForEachSecond);
    io:println(filterFailure is error, ":", filterFailureCount); // @output true:2
    io:println(forEachFailure is error, ":", forEachFailureCount); // @output true:2

    xml:Element shared = xml:createElement("shared", {}, xml``);
    xml:Element left = xml:createElement("left", {}, shared);
    xml:Element right = xml:createElement("right", {}, shared);
    io:println(left, right); // @output <left><shared/></left><right><shared/></right>
    error? indirectCycle = trap shared.setChildren(left);
    io:println(indirectCycle is error); // @output true
    io:println(shared); // @output <shared/>
    xml:Element atomicParent = xml:createElement("atomic", {}, xml `<old/>`);
    error? directCycle = trap atomicParent.setChildren(atomicParent);
    io:println(directCycle is error); // @output true
    io:println(atomicParent); // @output <atomic><old/></atomic>

    map<string> orderedBindings = {
        "{http://www.w3.org/2000/xmlns/}p": "urn:order",
        "{http://www.w3.org/2000/xmlns/}q": "urn:order"
    };
    xml:Element inheritedName = xml:createElement("{urn:order}x", {}, xml``);
    xml:Element inheritedRoot = xml:createElement("root", orderedBindings, inheritedName);
    io:println(inheritedRoot); // @output <root xmlns:p="urn:order" xmlns:q="urn:order"><p:x/></root>

    map<string> localBinding = {"{http://www.w3.org/2000/xmlns/}q": "urn:order"};
    xml:Element prunedChild = xml:createElement("{urn:order}x", localBinding, xml``);
    xml:Element prunedRoot = xml:createElement("root", orderedBindings, prunedChild);
    io:println(prunedRoot); // @output <root xmlns:p="urn:order" xmlns:q="urn:order"><p:x/></root>

    map<string> defaultFirst = {
        "{http://www.w3.org/2000/xmlns/}p": "urn:default",
        "{http://www.w3.org/2000/xmlns/}xmlns": "urn:default"
    };
    xml:Element defaultElement = xml:createElement("{urn:default}x", defaultFirst, xml``);
    io:println(defaultElement); // @output <x xmlns:p="urn:default" xmlns="urn:default"/>
    io:println(defaultElement.getAttributes()); // @output {"{http://www.w3.org/2000/xmlns/}p":"urn:default","{http://www.w3.org/2000/xmlns/}xmlns":"urn:default"}

    map<string> conflictingDefault = {"{http://www.w3.org/2000/xmlns/}xmlns": "urn:conflict"};
    error|xml:Element defaultFailure = trap xml:createElement("plain", conflictingDefault, xml``);
    io:println(defaultFailure is error); // @output true
    error? conflictingRename = trap defaultElement.setName("plain");
    io:println(conflictingRename is error); // @output true
    io:println(defaultElement.getName()); // @output {urn:default}x

    error|xml malformedElementFilter = trap (xml `<x/>`).elements("{bad");
    error|xml malformedChildFilter = trap (xml `<x><y/></x>`).elementChildren("{bad");
    io:println(malformedElementFilter is error); // @output true
    io:println(malformedChildFilter is error); // @output true

    error|xml:Element malformedName = trap xml:createElement("{}x", {}, xml``);
    error|xml:Element malformedBraceName = trap xml:createElement("{urn:{u}x", {}, xml``);
    error|xml:Comment trailingComment = trap xml:createComment("bad-");
    error|xml:ProcessingInstruction malformedTarget = trap xml:createProcessingInstruction("bad target", "x");
    error|xml:Text badText = trap xml:createText("\u{1}");
    io:println(malformedName is error); // @output true
    io:println(malformedBraceName is error); // @output true
    io:println(trailingComment is error); // @output true
    io:println(malformedTarget is error); // @output true
    io:println(badText is error); // @output true

    map<string> collisionBinding = {"{http://www.w3.org/2000/xmlns/}ns0": "urn:occupied"};
    map<string> generatedAttribute = {"{urn:generated}value": "v"};
    xml:Element generatedChild = xml:createElement("child", generatedAttribute, xml``);
    xml:Element collisionRoot = xml:createElement("root", collisionBinding,
        xml:concat(generatedChild, generatedChild));
    io:println(collisionRoot); // @output <root xmlns:ns0="urn:occupied"><child ns1:value="v" xmlns:ns1="urn:generated"/><child ns1:value="v" xmlns:ns1="urn:generated"/></root>

    // A pruned local declaration still reserves its lexical prefix for this element.
    map<string> inheritedNs0 = {"{http://www.w3.org/2000/xmlns/}p": "urn:shared"};
    map<string> prunedNs0AndAttribute = {
        "{urn:other}value": "v",
        "{http://www.w3.org/2000/xmlns/}ns0": "urn:shared"
    };
    xml:Element prunedCollisionChild = xml:createElement("{urn:shared}child", prunedNs0AndAttribute, xml``);
    xml:Element prunedCollisionRoot = xml:createElement("root", inheritedNs0, prunedCollisionChild);
    io:println(prunedCollisionRoot); // @output <root xmlns:p="urn:shared"><p:child ns1:value="v" xmlns:ns1="urn:other"/></root>

    map<string> defaultNamespace = {"{http://www.w3.org/2000/xmlns/}xmlns": "urn:reuse"};
    xml:Element defaultChild = xml:createElement("{urn:reuse}child", {}, xml``);
    xml:Element defaultRoot = xml:createElement("{urn:reuse}root", defaultNamespace, defaultChild);
    io:println(defaultRoot); // @output <root xmlns="urn:reuse"><child/></root>
    map<string> reservedAndEscaped = {
        "{http://www.w3.org/XML/1998/namespace}lang": "en",
        "plain": "a&\"b"
    };
    xml:Element escaped = xml:createElement("escaped", reservedAndEscaped, xml:createText("x<&"));
    io:println(escaped); // @output <escaped xml:lang="en" plain="a&amp;&quot;b">x&lt;&amp;</escaped>
    io:println(escaped); // @output <escaped xml:lang="en" plain="a&amp;&quot;b">x&lt;&amp;</escaped>
    io:println(escaped.getAttributes()); // @output {"{http://www.w3.org/XML/1998/namespace}lang":"en","plain":"a&\"b"}

    xml inheritedDefault = checkpanic xml:fromString("<root xmlns=\"urn:default-parent\"><child/></root>");
    xml:Element inheritedDefaultChild = <xml:Element>(<xml:Element>inheritedDefault).getChildren().get(0);
    io:println(inheritedDefaultChild.getName()); // @output {urn:default-parent}child
    io:println(inheritedDefaultChild.getAttributes().length()); // @output 0

    string nonXMLWhitespace = "\u{A0}";
    xml whitespaceValue = xml ` ${nonXMLWhitespace} `;
    xml strippedWhitespace = whitespaceValue.strip();
    io:println(strippedWhitespace.length()); // @output 1
    io:println(strippedWhitespace.data() == " " + nonXMLWhitespace + " "); // @output true

    map<string> badXMLBinding = {
        "{http://www.w3.org/2000/xmlns/}xml": "urn:not-xml"
    };
    map<string> badXMLNSBinding = {
        "{http://www.w3.org/2000/xmlns/}xmlns": "http://www.w3.org/2000/xmlns/"
    };
    error|xml:Element badXMLBindingResult = trap xml:createElement("x", badXMLBinding, xml``);
    error|xml:Element badXMLNSBindingResult = trap xml:createElement("x", badXMLNSBinding, xml``);
    io:println(badXMLBindingResult is error); // @output true
    io:println(badXMLNSBindingResult is error); // @output true

    string emptyInsertion = "";
    xml readonlyValue = xml `<readonly><child/>${emptyInsertion}</readonly>`;
    xml:Element & readonly readonlyElement = <xml:Element & readonly>readonlyValue;
    xml:Element & readonly readonlyChild = <xml:Element & readonly>readonlyElement.getChildren().get(0);
    error? readonlyNameFailure = trap readonlyElement.setName("changed");
    error? readonlyChildFailure = trap readonlyElement.setChildren("changed");
    error? nestedReadonlyFailure = trap readonlyChild.setName("changed");
    map<string> readonlySnapshot = readonlyElement.getAttributes();
    readonlySnapshot["safe"] = "yes";
    io:println(readonlyNameFailure is error); // @output true
    io:println(readonlyChildFailure is error); // @output true
    io:println(nestedReadonlyFailure is error); // @output true
    io:println(readonlyElement); // @output <readonly><child/></readonly>

    xml readonlyReachability = xml `<p:wrapper xmlns:p="urn:readonly" id="reachable"><nested/>${emptyInsertion}</p:wrapper>`;
    xml:Element & readonly reachableRoot = <xml:Element & readonly>readonlyReachability;
    xml:Element & readonly reachableChild = <xml:Element & readonly>reachableRoot.getChildren().get(0);
    error? reachableNameFailure = trap reachableRoot.setName("changed");
    error? reachableChildrenFailure = trap reachableChild.setChildren("changed");
    map<string> reachableSnapshot = reachableRoot.getAttributes();
    reachableSnapshot["id"] = "snapshot-only";
    io:println(reachableNameFailure is error); // @output true
    io:println(reachableChildrenFailure is error); // @output true
    io:println(readonlyReachability); // @output <p:wrapper id="reachable" xmlns:p="urn:readonly"><nested/></p:wrapper>
}
