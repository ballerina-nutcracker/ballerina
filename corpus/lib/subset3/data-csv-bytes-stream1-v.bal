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

import ballerina/data.csv;
import ballerina/io;

type Row record {|int a; string b;|};

type Rows Row[];

type Union (record {|int a;|}|record {|string a;|})[];

class FailingStream {
    private int calls = 0;

    public isolated function next() returns record {|byte[] value;|}|error? {
        self.calls += 1;
        if self.calls == 1 {
            return {value: [97, 44, 98, 10]};
        }
        return error("boom");
    }
}

public function main() {
    byte[] bytes = "a,b\n1,é".toBytes();
    io:println(csv:parseBytes(bytes, {}, Rows));
    io:println(csv:parseBytes(bytes, {encoding: "ISO-8859-1"}, Rows));
    io:println(csv:parseBytes("a,b\n1,\u{E9}".toBytes(), {encoding: "US-ASCII"}, Rows));
    io:println(csv:parseBytes([97, 44, 98, 10, 49, 44, 255], {encoding: "utf8"}, Rows));
    io:println(csv:parseBytes(bytes, {encoding: "FOO"}, Rows));

    // A multi-byte character split across two blocks.
    byte[] first = "a,b\n1,".toBytes();
    first.push(195);
    byte[][] blocks = [first, [169]];
    stream<byte[], error?> s1 = blocks.toStream();
    io:println(csv:parseStream(s1, {}, Rows));
    byte[][] bad = ["a,b\nz,x".toBytes()];
    stream<byte[], error?> s2 = bad.toStream();
    io:println(csv:parseStream(s2, {}, Rows));
    stream<byte[], error?> s3 = bad.toStream();
    io:println(csv:parseStream(s3, {encoding: "FOO"}, Rows));
    // An error from the source stream is returned as it is.
    stream<byte[], error?> s4 = new (new FailingStream());
    io:println(csv:parseStream(s4, {}, Rows));
    byte[][] unionBlocks = ["a\n1\nx".toBytes()];
    stream<byte[], error?> s5 = unionBlocks.toStream();
    io:println(csv:parseStream(s5, {}, Union));
}

// @output [{"a":1,"b":"é"}]
// @output [{"a":1,"b":"Ã©"}]
// @output [{"a":1,"b":"��"}]
// @output [{"a":1,"b":"�"}]
// @output error Error ("value '[97,44,98,10,49,44,195,169]' cannot be cast into 'typedesc {| a: int, b: string, never... |}[]'")
// @output [{"a":1,"b":"é"}]
// @output error Error ("Error occurred while reading the stream: value 'z' cannot be cast into 'int'")
// @output error Error ("Error occurred while reading the stream: FOO")
// @output error("boom")
// @output [{"a":"1"},{"a":"x"}]
