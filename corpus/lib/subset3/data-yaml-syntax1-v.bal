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

function show(anydata|error value) {
    if value is error {
        io:println(value.message());
    } else {
        io:println(value);
    }
}

function showText(string|error value) {
    if value is error {
        io:println(value.message());
    } else {
        io:println(value.length(), " ", value == "" ? "<empty>" : "ok");
    }
}

function text(string yamlText, string expected) {
    map<string>|error result = yaml:parseString(yamlText);
    if result is error {
        io:println(result.message());
        return;
    }
    io:println(result["a"] == expected);
}

public function main() {
    // Block and flow collections, comments and anchors.
    show(yaml:parseString("a:\n  b: 1 # comment\n  c: [1, 2, {d: e}]\nf:\n- x\n- y\n# trailing"));
    show(yaml:parseString("- - 1\n  - 2\n- k: v\n  l: w\n-\n- [a, b: c]"));
    show(yaml:parseString("a: &x\n  p: 1\nb: *x\nc: &y [1, 2]\nd: *y\ne: &z {q: 1}\nf: *z"));
    show(yaml:parseString("a: &x 1\nb: &x 2\nc: *x"));
    show(yaml:parseString("a: &x 1\nb: &x 2\nc: *x", {allowAnchorRedefinition: false}));
    show(yaml:parseString("a: *missing"));
    show(yaml:parseString("a: &x !!int *x"));
    show(yaml:parseString("a: http://x.com/a#b\nb: 12:30:00\nc: value#not-comment\nd: -1\ne: -x"));
    show(yaml:parseString("a: 1\r\nb: 2\r\n"));
    show(yaml:parseString("{\"a\":1, 'b':[2], \"c\":{\"d\":3}}"));
    show(yaml:parseString("a: !<tag:yaml.org,2002:str> x\nb: !<tag:yaml.org,2002:%69nt> 5"));
    show(yaml:parseString("--- a"));
    show(yaml:parseString("a: 1\n...\nb: 2"));
    // Scalar styles fold and chomp as YAML specifies.
    text("a: \"x\n  y\"", "x y");
    text("a: \"x\n\n  y\"", "x\ny");
    text("a: \"x \\\n  y\"", "x y");
    text("a: 'x\n\n  y'", "x\ny");
    text("a: 'it''s'", "it's");
    text("a: \"a\\\\b\\t\\x41\\u00e9\\U0001F600\\/\\_\"", "a\\b\tA\u{e9}\u{1F600}/\u{a0}");
    text("a: plain\n  multi\n\n  line", "plain multi\nline");
    text("a: |\n  line1\n  line2\n", "line1\nline2\n");
    text("a: |-\n  x\n", "x");
    text("a: |+\n  x\n\n\nb: 1", "x\n\n\n");
    text("a: |\n\n  x\n", "\nx\n");
    text("a: |2\n   x\n", " x\n");
    text("a: >\n  folded\n  text\n\n  para\n    more\n  back\n", "folded text\npara\n  more\nback\n");
    text("a: | # comment\n  x\n# trailing\nb: c", "x\n");
    text("a: |\n  x", "x\n");
    showText(yaml:parseString("? |\n  block key\n: v"));
    // Syntax errors are reported with their position.
    show(yaml:parseString("[1, 2"));
    show(yaml:parseString("a: \"x"));
    show(yaml:parseString("a: 'x"));
    show(yaml:parseString("a: \"\\q\""));
    show(yaml:parseString("a: \"\\u00zz\""));
    show(yaml:parseString("a: 1\n  b: 2"));
    show(yaml:parseString("a:\n  b: 1\n c: 2"));
    show(yaml:parseString("a:\n\tb: 1"));
    show(yaml:parseString("a: b: c"));
    show(yaml:parseString("a: - x"));
    show(yaml:parseString("[- x]"));
    show(yaml:parseString("[a: |\n  x]"));
    show(yaml:parseString("--- a: 1"));
    show(yaml:parseString("a: 1\n... b"));
    show(yaml:parseString("a: |x\n  y"));
    show(yaml:parseString("@a"));
    show(yaml:parseString("a: x\u{1}y"));
    show(yaml:parseString("a\nb: 1"));
    show(yaml:parseString("[a, b]: 1"));
    show(yaml:parseString("a: b # c\n  d: e"));
}

// @output {"a":{"b":1,"c":[1,2,{"d":"e"}]},"f":["x","y"]}
// @output [[1,2],{"k":"v","l":"w"},null,["a",{"b":"c"}]]
// @output {"a":{"p":1},"b":{"p":1},"c":[1,2],"d":[1,2],"e":{"q":1},"f":{"q":1}}
// @output {"a":1,"b":2,"c":2}
// @output 'duplicate anchor definition' at line: '3' column: '0'
// @output 'anchor does not exist' at line: '1' column: '10'
// @output 'an alias node cannot have tag properties' at line: '1' column: '13'
// @output {"a":"http://x.com/a#b","b":"12:30:00","c":"value#not-comment","d":-1,"e":"-x"}
// @output {"a":1,"b":2}
// @output {"a":1,"b":[2],"c":{"d":3}}
// @output {"a":"x","b":5}
// @output a
// @output [{"a":1},{"b":2}]
// @output true
// @output true
// @output true
// @output true
// @output true
// @output true
// @output true
// @output true
// @output true
// @output true
// @output true
// @output true
// @output true
// @output true
// @output true
// @output invalid type 'string' expected 'map type'
// @output 'unexpected event' at line: '1' column: '4'
// @output 'invalid token 'EOL' inside the double-quoted scalar' at line: '1' column: '4'
// @output 'END of stream reached' at line: '1' column: '4'
// @output 'invalid escape character' at line: '1' column: '5'
// @output 'expected a unicode character after escaped char' at line: '1' column: '8'
// @output Cannot have nested mapping under a key-pair that is already assigned
// @output 'invalid indentation' at line: '3' column: '2'
// @output 'cannot have tab as an indentation' at line: '2' column: '2'
// @output 'two block mapping keys cannot be defined in the same line' at line: '1' column: '5'
// @output 'cannot have nested sequence for a defined value' at line: '1' column: '4'
// @output 'cannot have block sequence under flow collection' at line: '1' column: '2'
// @output 'cannot have a block node inside a flow node' at line: '1' column: '5'
// @output 'block collection token cannot start in the same line as the directive marker' at line: '1' column: '5'
// @output 'token cannot start in the same line as the document marker' at line: '2' column: '4'
// @output 'invalid block header' at line: '1' column: '4'
// @output 'invalid yaml document' at line: '1' column: '0'
// @output 'non printable character found' at line: '1' column: '4'
// @output 'there can only be one root event to a document' at line: '2' column: '3'
// @output 'there can only be one root event to a document' at line: '1' column: '8'
// @output Cannot have nested mapping under a key-pair that is already assigned
