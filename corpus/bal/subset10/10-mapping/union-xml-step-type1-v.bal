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

type Elements record {|
    xml<xml:Element> v;
    int m;
|};

type Texts record {|
    xml<xml:Text> v;
    int m;
|};

type Any record {|
    xml v;
    int m;
|};

function kind(any v) returns string {
    if v is Elements {
        return "Elements";
    }
    if v is Texts {
        return "Texts";
    }
    if v is Any {
        return "Any";
    }
    if v is [xml<xml:Element>, int] {
        return "[xml<xml:Element>, int]";
    }
    if v is [xml, int] {
        return "[xml, int]";
    }
    if v is [function (xml) returns xml<xml:Element>, int] {
        return "[function (xml) returns xml<xml:Element>, int]";
    }
    return "other";
}

// The step's own result type decides the candidate: a filtered step is xml<xml:Element>, an unfiltered one is xml.
public function main() {
    xml x = xml `<root><a>1</a>text<a>2</a></root>`;
    Elements|Texts filtered = {v: x/<a>, m: 1};
    io:println(kind(filtered)); // @output Elements
    io:println(filtered); // @output {"v":<a>1</a><a>2</a>,"m":1}
    Texts|Any unfiltered = {v: x/*, m: 1};
    io:println(kind(unfiltered)); // @output Any
    io:println(unfiltered); // @output {"v":<a>1</a>text<a>2</a>,"m":1}
    [xml<xml:Element>, int]|[xml, int] l = [x/*, 1];
    io:println(kind(l)); // @output [xml, int]
    io:println(l); // @output [<a>1</a>text<a>2</a>,1]
    [function (xml) returns xml<xml:Text>, int]|[function (xml) returns xml<xml:Element>, int] lambda = [s => s/<a>, 1];
    io:println(kind(lambda)); // @output [function (xml) returns xml<xml:Element>, int]
    [function (xml) returns xml<xml:Text>, int]|[function (xml) returns xml<xml:Element>, int] indexed = [s => s/<a>[0], 1];
    io:println(kind(indexed)); // @output [function (xml) returns xml<xml:Element>, int]
    if indexed is [function (xml) returns xml<xml:Element>, int] {
        var f = indexed[0];
        io:println(f(x)); // @output <a>1</a>
    }
}
