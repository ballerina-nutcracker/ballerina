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

type Person record {|
    string name;
    int age;
|};

class FailingStream {
    private int calls = 0;

    public isolated function next() returns record {|byte[] value;|}|error? {
        self.calls += 1;
        if self.calls == 1 {
            return {value: [97, 58, 32, 49, 10]};
        }
        return error("boom");
    }
}

function show(anydata|error value) {
    if value is error {
        io:println(value.message());
    } else {
        io:println(value);
    }
}

public function main() {
    map<string>|yaml:Error fromBytes = yaml:parseBytes("name: café".toBytes());
    show(fromBytes);
    map<string>|yaml:Error withBom = yaml:parseBytes([239, 187, 191, 97, 58, 32, 98]);
    show(withBom);
    map<string>|yaml:Error invalid = yaml:parseBytes([97, 58, 32, 255]);
    show(invalid);
    // Each malformed sequence becomes one U+FFFD, grouped as Java's decoder groups it.
    map<string>|yaml:Error malformed = yaml:parseBytes([97, 58, 32, 255, 254, 226, 130, 65, 194, 66, 224, 128, 67, 237, 160, 128, 68, 240, 128, 69, 244, 144, 70, 241, 128, 128, 71]);
    show(malformed);

    // A multi-byte character split across two blocks.
    byte[] first = "name: caf".toBytes();
    first.push(195);
    byte[][] blocks = [first, [169, 10, 97, 103, 101, 58, 32, 51]];
    Person|yaml:Error person = yaml:parseStream(blocks.toStream());
    show(person);

    // Binding errors from a stream carry a prefix; syntax errors do not.
    byte[][] missing = ["name: x".toBytes()];
    Person|yaml:Error missingAge = yaml:parseStream(missing.toStream());
    show(missingAge);
    byte[][] syntax = ["[1, 2".toBytes()];
    int[]|yaml:Error syntaxError = yaml:parseStream(syntax.toStream());
    show(syntaxError);
    byte[][] none = [];
    map<int>|yaml:Error empty = yaml:parseStream(none.toStream());
    show(empty);

    // An error from the source stream is returned.
    stream<byte[], error?> failing = new (new FailingStream());
    map<int>|yaml:Error streamError = yaml:parseStream(failing);
    show(streamError);
    show(streamError is yaml:Error);
}

// @output {"name":"café"}
// @output {"a":"b"}
// @output {"a":"�"}
// @output {"a":"���A�B��C�D��E��F�G"}
// @output {"name":"café","age":3}
// @output Error occurred while reading the stream: required field 'age' not present in YAML
// @output 'unexpected event' at line: '1' column: '4'
// @output 'invalid yaml document' at line: '1' column: '0'
// @output Error occurred while reading the stream: boom
// @output true
