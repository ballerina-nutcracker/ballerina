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

public function main() returns error? {
    string path = "/tmp/bal_io_char_props.properties";
    string content = "# a comment\n! another comment\nname=Ballerina\n  spaced.key : value with spaces  \nempty=\nesc\\:aped=x\nmulti=line one \\\n    continued\ntabbed\tvalue\nuni=\\u00E9clair\nnl=a\\nb\n";
    check io:fileWriteString(path, content);

    io:ReadableByteChannel rb = check io:openReadableFile(path);
    io:ReadableCharacterChannel rc = new (rb, "UTF-8");
    io:println(check rc.readProperty("name")); // @output Ballerina
    io:println("[", check rc.readProperty("spaced.key"), "]"); // @output [value with spaces  ]
    io:println(check rc.readProperty("missing", "fallback")); // @output fallback
    io:println(check rc.readProperty("esc:aped")); // @output x
    io:println(check rc.readProperty("multi")); // @output line one continued
    io:println(check rc.readProperty("tabbed")); // @output value
    io:println(check rc.readProperty("uni")); // @output éclair
    string nl = check rc.readProperty("nl");
    io:println(nl.length()); // @output 3
    map<string> all = check rc.readAllProperties();
    io:println(all.length()); // @output 8
    io:println(all["empty"] == ""); // @output true
    check rc.close();

    // writeProperties roundtrip: the output carries a timestamp header, so
    // assert by reading the properties back rather than comparing raw text
    string wpath = "/tmp/bal_io_char_props_out.properties";
    io:WritableByteChannel wb = check io:openWritableFile(wpath);
    io:WritableCharacterChannel wc = new (wb, "UTF-8");
    check wc.writeProperties({"key one": "value=1", "plain": "text", "uni": "ü", "tab": "a\tb", "line": "a\nb"}, "generated");
    check wc.close();

    io:ReadableByteChannel vb = check io:openReadableFile(wpath);
    io:ReadableCharacterChannel vc = new (vb, "UTF-8");
    map<string> written = check vc.readAllProperties();
    io:println(written.length()); // @output 5
    io:println(written["key one"]); // @output value=1
    io:println(written["plain"]); // @output text
    io:println(written["uni"]); // @output ü
    io:println(written["tab"] == "a\tb"); // @output true
    io:println(written["line"] == "a\nb"); // @output true
    check vc.close();
}
