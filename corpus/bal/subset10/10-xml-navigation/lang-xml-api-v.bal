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

function duplicate(xml:Element|xml:Text|xml:Comment|xml:ProcessingInstruction item) returns xml {
    return item + item;
}

function isElement(xml:Element|xml:Text|xml:Comment|xml:ProcessingInstruction item) returns boolean {
    return item is xml:Element;
}

function ignore(xml:Element|xml:Text|xml:Comment|xml:ProcessingInstruction item) {
    _ = item;
}

function elementIdentity(xml:Element item) returns xml:Element {
    return item;
}

function commentIdentity(xml:Comment item) returns xml:Comment {
    return item;
}

function piIdentity(xml:ProcessingInstruction item) returns xml:ProcessingInstruction {
    return item;
}

function textIdentity(xml:Text item) returns xml:Text {
    return item;
}

function keepComment(xml:Comment item) returns boolean {
    return item.getContent() == "narrow-comment";
}

function keepPI(xml:ProcessingInstruction item) returns boolean {
    return item.getTarget() == "narrow-pi";
}

function keepText(xml:Text item) returns boolean {
    return item.data() == "narrow-text";
}

function visitComment(xml:Comment item) {
    io:println("comment:", item.getContent());
}

function visitPI(xml:ProcessingInstruction item) {
    io:println("pi:", item.getTarget());
}

function visitText(xml:Text item) {
    io:println("text:", item.data());
}

int visitNumber = 0;

function recordVisit(xml:Element|xml:Text item) {
    visitNumber += 1;
    io:println(visitNumber, ":", item);
}

public function main() returns error? {
    io:println(xml:XML_NAMESPACE_URI); // @output http://www.w3.org/XML/1998/namespace
    io:println(xml:XMLNS_NAMESPACE_URI); // @output http://www.w3.org/2000/xmlns/
    io:println(xml:space); // @output {http://www.w3.org/XML/1998/namespace}space
    io:println(xml:lang); // @output {http://www.w3.org/XML/1998/namespace}lang
    io:println(xml:base); // @output {http://www.w3.org/XML/1998/namespace}base

    xml mixed = xml `<a>one<b>two</b><!--c--><?p d?></a>tail`;
    io:println(mixed.length()); // @output 2
    io:println(xml:length(mixed)); // @output 2
    io:println(mixed.get(0)); // @output <a>one<b>two</b><!--c--><?p d?></a>
    io:println(xml:get(mixed, 1)); // @output tail
    io:println(mixed.slice(0, 1)); // @output <a>one<b>two</b><!--c--><?p d?></a>
    io:println(xml:slice(mixed, 1, 2)); // @output tail
    io:println(xml:concat("x", xml `<c/>`, "y")); // @output x<c/>y
    io:println(xml:concat().length()); // @output 0
    io:println(xml:concat("", "").length()); // @output 0
    io:println(mixed.slice(1, 1).length()); // @output 0
    io:println(mixed.slice(0, 0).length()); // @output 0
    io:println(mixed.slice(mixed.length(), mixed.length()).length()); // @output 0
    io:println(mixed.slice(0, mixed.length())); // @output <a>one<b>two</b><!--c--><?p d?></a>tail
    xml threeItems = xml `<first/><middle/><last/>`;
    io:println(threeItems.slice(1, 2)); // @output <middle/>

    xml:Element exactSource = xml `<exact/>`;
    xml:Element exactGet = exactSource.get(0);
    xml<xml:Element> exactSlice = exactSource.slice(0, 1);
    xml<xml:Element> exactMap = exactSource.map(elementIdentity);
    io:println(exactGet, exactSlice, exactMap); // @output <exact/><exact/><exact/>
    io:println(xml:concat(exactSource) === exactSource); // @output true
    var elementNext = exactSource.iterator().next();
    var commentNext = (xml `<!--i-->`).iterator().next();
    var piNext = (xml `<?i data?>`).iterator().next();
    var textNext = (xml `text`).iterator().next();
    io:println(elementNext is record {| xml:Element value; |}); // @output true
    io:println(commentNext is record {| xml:Comment value; |}); // @output true
    io:println(piNext is record {| xml:ProcessingInstruction value; |}); // @output true
    io:println(textNext is record {| xml:Text value; |}); // @output true
    var exhausted = exactSource.iterator();
    _ = exhausted.next();
    io:println(exhausted.next() is ()); // @output true

    // Opaque functions are available as both module calls and methods.
    var moduleIterator = xml:iterator(exactSource);
    xml:Element moduleGet = xml:get(exactSource, 0);
    xml<xml:Element> moduleSlice = xml:slice(exactSource, 0, 1);
    xml<xml:Element> moduleMap = xml:map(exactSource, elementIdentity);
    xml:forEach(exactSource, ignore);
    // Unlike its callback parameter, filter's declared result deliberately remains broad xml.
    xml moduleFilter = xml:filter(exactSource, isElement);
    io:println(moduleIterator.next() is record {| xml:Element value; |}, moduleGet,
        moduleSlice, moduleMap, moduleFilter); // @output true<exact/><exact/><exact/><exact/>

    // Each generic opaque API preserves narrowed constituent input and callback types.
    xml:Comment narrowComment = xml `<!--narrow-comment-->`;
    xml:Comment commentGet = narrowComment.get(0);
    xml<xml:Comment> commentSlice = narrowComment.slice(0, 1);
    xml<xml:Comment> commentMap = narrowComment.map(commentIdentity);
    narrowComment.forEach(visitComment); // @output comment:narrow-comment
    xml commentFilter = narrowComment.filter(keepComment);
    io:println(commentGet === narrowComment, commentSlice, commentMap,
        commentFilter); // @output true<!--narrow-comment--><!--narrow-comment--><!--narrow-comment-->

    xml:ProcessingInstruction narrowPI = xml `<?narrow-pi value?>`;
    xml:ProcessingInstruction piGet = narrowPI.get(0);
    xml<xml:ProcessingInstruction> piSlice = narrowPI.slice(0, 1);
    xml<xml:ProcessingInstruction> piMap = narrowPI.map(piIdentity);
    narrowPI.forEach(visitPI); // @output pi:narrow-pi
    xml piFilter = narrowPI.filter(keepPI);
    io:println(piGet === narrowPI, piSlice, piMap,
        piFilter); // @output true<?narrow-pi value?><?narrow-pi value?><?narrow-pi value?>

    xml:Text narrowText = xml `narrow-text`;
    xml:Text textGet = narrowText.get(0);
    xml<xml:Text> textSlice = narrowText.slice(0, 1);
    xml<xml:Text> textMap = narrowText.map(textIdentity);
    narrowText.forEach(visitText); // @output text:narrow-text
    xml textFilter = narrowText.filter(keepText);
    io:println(textGet, "|", textSlice, "|", textMap,
        "|", textFilter); // @output narrow-text|narrow-text|narrow-text|narrow-text

    xml:Element elem = <xml:Element>mixed.get(0);
    io:println(elem.getName()); // @output a
    elem.setName("{urn:new}renamed");
    io:println(elem); // @output <renamed xmlns="urn:new">one<b xmlns="">two</b><!--c--><?p d?></renamed>
    elem.setName("a");
    io:println(elem.getChildren()); // @output one<b>two</b><!--c--><?p d?>
    io:println(elem.getDescendants()); // @output one<b>two</b>two<!--c--><?p d?>
    io:println(elem.data()); // @output onetwo

    // Normally expressible functions are also exercised through module calls.
    xml:Element moduleElem = xml:createElement("module", {}, xml `<inner>value</inner>`);
    io:println(xml:getName(moduleElem)); // @output module
    xml:setName(moduleElem, "module2");
    io:println(xml:getAttributes(moduleElem).length()); // @output 0
    io:println(xml:getChildren(moduleElem)); // @output <inner>value</inner>
    xml:setChildren(moduleElem, xml `<replacement>module-data</replacement>`);
    io:println(xml:getDescendants(moduleElem)); // @output <replacement>module-data</replacement>module-data
    io:println(xml:data(moduleElem)); // @output module-data
    xml:Comment moduleComment = xml:createComment("module-comment");
    xml:ProcessingInstruction modulePI = xml:createProcessingInstruction("module", "pi");
    io:println(xml:getContent(moduleComment)); // @output module-comment
    io:println(xml:getTarget(modulePI), ":", xml:getContent(modulePI)); // @output module:pi
    io:println(xml:createText("module-text")); // @output module-text

    map<string> sourceAttrs = {"id": "1", "{urn:attr}code": "x",
        "{http://www.w3.org/2000/xmlns/}p": "urn:p"};
    xml:Element made = xml:createElement("{urn:made}root", sourceAttrs, xml `<child/>`);
    sourceAttrs["id"] = "changed";
    io:println(made); // @output <root id="1" ns0:code="x" xmlns:p="urn:p" xmlns:ns0="urn:attr" xmlns="urn:made"><child xmlns=""/></root>
    map<string> snapshot1 = made.getAttributes();
    map<string> snapshot2 = made.getAttributes();
    snapshot1["id"] = "snapshot";
    io:println(snapshot1 === snapshot2); // @output false
    io:println(snapshot2); // @output {"id":"1","{urn:attr}code":"x","{http://www.w3.org/2000/xmlns/}p":"urn:p"}
    io:println(made); // @output <root id="1" ns0:code="x" xmlns:p="urn:p" xmlns:ns0="urn:attr" xmlns="urn:made"><child xmlns=""/></root>
    // The default declaration is synthesized only for serialization.
    xml:Element namespacedEmpty = xml:createElement("{urn:empty}root", {}, xml``);
    io:println(namespacedEmpty.getAttributes().length()); // @output 0
    io:println(namespacedEmpty); // @output <root xmlns="urn:empty"/>
    made.setChildren("changed");
    io:println(made.getChildren()); // @output changed
    // Empty text normalizes to empty XML; see ballerina-lang#44701.
    made.setChildren("");
    io:println(made.getChildren().length()); // @output 0
    made.setChildren(xml `<xmlChild/>`);
    io:println(made.getChildren()); // @output <xmlChild/>

    xml:Comment comment = xml:createComment("note");
    xml:ProcessingInstruction pi = xml:createProcessingInstruction("target", "content");
    io:println(comment.getContent()); // @output note
    io:println(pi.getTarget()); // @output target
    io:println(pi.getContent()); // @output content
    // createText("") is empty XML, unlike ballerina-lang#44701.
    xml:Text empty = xml:createText("");
    io:println(empty.length()); // @output 0

    xml selected = xml `<x/> t <!--drop--><?drop p?><y><z/></y>`;
    io:println(selected.strip()); // @output <x/> t <y><z/></y>
    io:println(selected.elements()); // @output <x/><y><z/></y>
    io:println(selected.elements("y")); // @output <y><z/></y>
    io:println(selected.children()); // @output <z/>
    io:println(selected.elementChildren()); // @output <z/>
    io:println("[", selected.text(), "]"); // @output [ t ]
    io:println(xml:strip(selected)); // @output <x/> t <y><z/></y>
    io:println(xml:elements(selected)); // @output <x/><y><z/></y>
    io:println(xml:children(selected)); // @output <z/>
    io:println(xml:elementChildren(selected)); // @output <z/>
    io:println("[", xml:text(selected), "]"); // @output [ t ]

    xml nestedOrder = xml `<root><nested><deep/></nested><sibling/></root>`;
    io:println((<xml:Element>nestedOrder).getDescendants()); // @output <nested><deep/></nested><deep/><sibling/>
    xml shallowWhitespace = xml ` <x> </x> `;
    io:println("[", shallowWhitespace.strip(), "]"); // @output [<x> </x>]

    xml mapped = mixed.map(duplicate);
    io:println(mapped); // @output <a>one<b>two</b><!--c--><?p d?></a><a>one<b>two</b><!--c--><?p d?></a>tailtail
    mixed.forEach(ignore);
    xml filtered = mixed.filter(isElement);
    io:println(filtered === mixed.get(0)); // @output true
    xml<xml:Element|xml:Text> orderedCallbacks = xml `<a/>text<b/>`;
    orderedCallbacks.forEach(recordVisit); // @output 1:<a/>
                                               // @output 2:text
                                               // @output 3:<b/>
    xml normalizedMap = (xml `a`).map(duplicate);
    io:println(normalizedMap.length(), ":", normalizedMap); // @output 1:aa

    xml firstSource = xml `<first/>a`;
    xml secondSource = xml `b<second/>`;
    xml combined = xml:concat(firstSource, secondSource);
    io:println(combined); // @output <first/>ab<second/>
    io:println(firstSource, "|", secondSource); // @output <first/>a|b<second/>

    xml commentItem = xml `<!--identity-->`;
    xml piItem = xml `<?identity value?>`;
    io:println(commentItem.get(0) === commentItem); // @output true
    io:println(piItem.get(0) === piItem); // @output true

    xml parsed = check xml:fromString(" before <!--c--><p:r xmlns:p=\"urn:p\">&amp;<![CDATA[x]]></p:r><?q d?> after ");
    io:println("[", parsed, "]"); // @output [ before <!--c--><p:r xmlns:p="urn:p">&amp;x</p:r><?q d?> after ]
    io:println(parsed.length()); // @output 5

    // A long mixed sequence guards the linear normalization/traversal paths.
    xml longMixed = xml `<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t<n/>t`;
    io:println(longMixed.length()); // @output 48
    xml repeatedConcat = xml``;
    foreach int _ in 0 ..< 1000 {
        repeatedConcat = repeatedConcat + xml `<n/>`;
        repeatedConcat = repeatedConcat + xml `t`;
    }
    io:println(repeatedConcat.length()); // @output 2000
    xml branchBase = xml `<base1/><base2/>`;
    xml leftBranch = branchBase + xml `<left/>`;
    xml rightBranch = branchBase + xml `<right/>`;
    io:println(branchBase); // @output <base1/><base2/>
    io:println(leftBranch); // @output <base1/><base2/><left/>
    io:println(rightBranch); // @output <base1/><base2/><right/>
}
