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

import ballerina/data.yaml;
import ballerina/io;

type Book record {|
    string title;
    int year;
    string[] tags;
|};

function show(string|error value) {
    if value is error {
        io:println(value.message());
    } else {
        io:println(value);
        io:println("--");
    }
}

public function main() {
    map<anydata> library = {
        name: "Central Library",
        books: [{title: "1984", year: 1949, tags: ["dystopia"]}, {title: "Emma", year: 1815, tags: []}],
        open: true,
        rating: 4.5,
        fee: 2.50d,
        empty: {}
    };
    show(yaml:toYamlString(library));
    show(yaml:toYamlString(library, {indentationPolicy: 4}));
    show(yaml:toYamlString(library, {flowStyle: true}));
    show(yaml:toYamlString(library, {canonical: true}));
    show(yaml:toYamlString(library, {canonical: true, flowStyle: true}));
    show(yaml:toYamlString({a: "x", b: 1}, {forceQuotes: true}));
    show(yaml:toYamlString({a: "it's", b: 1}, {forceQuotes: true, useSingleQuotes: true}));
    show(yaml:toYamlString([1, "a", {b: 2}, [3]], {isStream: true}));
    show(yaml:toYamlString([], {isStream: true}));
    show(yaml:toYamlString({a: [1, 2]}, {isStream: true}));
    show(yaml:toYamlString([]));
    show(yaml:toYamlString({}));
    show(yaml:toYamlString("Simple Scalar"));
    show(yaml:toYamlString([[1, 2], [3]]));
    Book book = {title: "Emma", year: 1815, tags: ["classic"]};
    show(yaml:toYamlString(book));
    // Strings that would not read back as the same string are quoted, and nil is written as null.
    show(yaml:toYamlString({a: "", b: "true", c: "123", d: "a: b", e: "- x", f: "a #b", g: "line1\nline2", h: " x", i: "say \"hi\"", j: ()}));
    show(yaml:toYamlString([1.0, 1.0e20, 1.0e-5, -0.0, 1.5e300]));
    show(yaml:toYamlString({a: 1}, {indentationPolicy: -1}));
    show(yaml:toYamlString("..."));
    show(yaml:toYamlString(["a,b", "c]d", {"k,1": "v{2}"}], {flowStyle: true}));
    show(yaml:toYamlString({a: 1, l: [1, 2]}, {indentationPolicy: 0}));
    show(yaml:toYamlString({parent: {child: 1}}, {indentationPolicy: 0}));
    show(yaml:toYamlString({a: {}}, {indentationPolicy: 0}));
    show(yaml:toYamlString([{}], {indentationPolicy: 0}));
}

// @output name: Central Library
// @output books:
// @output -
// @output   title: "1984"
// @output   year: 1949
// @output   tags:
// @output   - dystopia
// @output -
// @output   title: Emma
// @output   year: 1815
// @output   tags: []
// @output open: true
// @output rating: 4.5
// @output fee: 2.50
// @output empty: {}
// @output --
// @output name: Central Library
// @output books:
// @output -
// @output     title: "1984"
// @output     year: 1949
// @output     tags:
// @output     - dystopia
// @output -
// @output     title: Emma
// @output     year: 1815
// @output     tags: []
// @output open: true
// @output rating: 4.5
// @output fee: 2.50
// @output empty: {}
// @output --
// @output {name: Central Library, books: [{title: "1984", year: 1949, tags: [dystopia]}, {title: Emma, year: 1815, tags: []}], open: true, rating: 4.5, fee: 2.50, empty: {}}
// @output --
// @output !!str name: !!str Central Library
// @output !!str books: !!seq
// @output -
// @output   !!str title: !!str "1984"
// @output   !!str year: !!str 1949
// @output   !!str tags: !!seq
// @output   - !!str dystopia
// @output -
// @output   !!str title: !!str Emma
// @output   !!str year: !!str 1815
// @output   !!str tags: !!seq []
// @output !!str open: !!str true
// @output !!str rating: !!str 4.5
// @output !!str fee: !!str 2.50
// @output !!str empty: {}
// @output --
// @output {!!str name: !!str Central Library, !!str books: !!seq [{!!str title: !!str "1984", !!str year: !!str 1949, !!str tags: !!seq [!!str dystopia]}, {!!str title: !!str Emma, !!str year: !!str 1815, !!str tags: !!seq []}], !!str open: !!str true, !!str rating: !!str 4.5, !!str fee: !!str 2.50, !!str empty: {}}
// @output --
// @output "a": "x"
// @output "b": "1"
// @output --
// @output 'a': 'it''s'
// @output 'b': '1'
// @output --
// @output 1
// @output ---
// @output a
// @output ---
// @output b: 2
// @output ---
// @output - 3
// @output --
// @output 
// @output --
// @output a:
// @output - 1
// @output - 2
// @output --
// @output []
// @output --
// @output {}
// @output --
// @output Simple Scalar
// @output --
// @output -
// @output   - 1
// @output   - 2
// @output -
// @output   - 3
// @output --
// @output title: Emma
// @output year: 1815
// @output tags:
// @output - classic
// @output --
// @output a: ""
// @output b: "true"
// @output c: "123"
// @output d: "a: b"
// @output e: "- x"
// @output f: "a #b"
// @output g: "line1\nline2"
// @output h: " x"
// @output i: say "hi"
// @output j: null
// @output --
// @output - 1.0
// @output - 1.0E20
// @output - 1.0E-5
// @output - -0.0
// @output - 1.5E300
// @output --
// @output count is negative: -1
// @output "..."
// @output --
// @output ["a,b", "c]d", {"k,1": "v{2}"}]
// @output --
// @output a: 1
// @output l:
// @output - 1
// @output - 2
// @output --
// @output indentationPolicy must be greater than 0 to write a nested block collection
// @output a: {}
// @output --
// @output - {}
// @output --
