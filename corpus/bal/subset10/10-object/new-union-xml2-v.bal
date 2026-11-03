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

class IntArg {
    xml[] values;
    int marker;

    function init(xml children, xml all, xml descendants, xml indexed, xml filtered, xml called, int marker) {
        self.values = [children, all, descendants, indexed, filtered, called];
        self.marker = marker;
    }
}

class StringArg {
    xml[] values;
    string marker;

    function init(xml children, xml all, xml descendants, xml indexed, xml filtered, xml called, string marker) {
        self.values = [children, all, descendants, indexed, filtered, called];
        self.marker = marker;
    }
}

function kind(any v) returns string {
    return v is IntArg ? "IntArg" : v is StringArg ? "StringArg" : "neither";
}

public function main() {
    xml x = xml `<root><a>1<b>x</b></a><a>2</a><c><a>3</a></c></root>`;
    IntArg|StringArg o = new (x/<a>, x/*, x/**/<a>, x/<a>[1], x/*.<a>, x/<a>.get(0), 1);
    io:println(kind(o)); // @output IntArg
    if o is IntArg {
        io:println(o.values); // @output [<a>1<b>x</b></a><a>2</a>,<a>1<b>x</b></a><a>2</a><c><a>3</a></c>,<a>1<b>x</b></a><a>2</a><a>3</a>,<a>2</a>,<a>1<b>x</b></a><a>2</a>,<a>1<b>x</b></a>]
    }
    IntArg|StringArg p = new (x/<a>, x/*, x/**/<a>, x/<a>[1], x/*.<a>, x/<a>.get(0), "s");
    io:println(kind(p)); // @output StringArg
    if p is StringArg {
        io:println(p.values); // @output [<a>1<b>x</b></a><a>2</a>,<a>1<b>x</b></a><a>2</a><c><a>3</a></c>,<a>1<b>x</b></a><a>2</a><a>3</a>,<a>2</a>,<a>1<b>x</b></a><a>2</a>,<a>1<b>x</b></a>]
    }
}
