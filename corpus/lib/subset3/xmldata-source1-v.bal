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

import ballerina/data.xmldata;
import ballerina/io;

type Person record {|
    string name;
    int age;
|};

class FailingBlocks {
    int count = 0;

    public isolated function next() returns record {|byte[] value;|}|error? {
        self.count += 1;
        if self.count == 1 {
            return {value: "<p><name>A</name>".toBytes()};
        }
        return error("stream failure");
    }
}

class FailingAfterDocument {
    int count = 0;

    public isolated function next() returns record {|byte[] value;|}|error? {
        self.count += 1;
        if self.count == 1 {
            return {value: "<p><name>A</name><age>1</age></p>".toBytes()};
        }
        return error("stream failure");
    }
}

type JsonValue record {|
    json value;
|};

function chunk(byte[] bytes, int 'start, int end) returns byte[] {
    byte[] result = [];
    foreach int i in 'start ..< end {
        result.push(bytes[i]);
    }
    return result;
}

function show(anydata|error v) {
    if v is error {
        io:println("E: ", v.message());
    } else {
        io:println(v);
    }
}

public function main() {
    byte[] personBytes = string `<p><name>Ké</name><age>40</age></p>`.toBytes();
    Person|xmldata:Error b1 = xmldata:parseBytes(personBytes);
    show(b1);
    // @output {"name":"Ké","age":40}
    Person|xmldata:Error b2 = xmldata:parseBytes(string `<p><name>😀</name><age>1</age></p>`.toBytes());
    show(b2);
    // @output {"name":"😀","age":1}

    stream<byte[], error?> s1 = [chunk(personBytes, 0, 13), chunk(personBytes, 13, personBytes.length())].toStream();
    Person|xmldata:Error r1 = xmldata:parseStream(s1);
    show(r1);
    // @output {"name":"Ké","age":40}
    stream<byte[], error?> s2 = ["<p><name>B</name>".toBytes(), "<age>2</age>".toBytes(), "</p>".toBytes()].toStream();
    Person|xmldata:Error r2 = xmldata:parseStream(s2);
    show(r2);
    // @output {"name":"B","age":2}
    stream<byte[], error?> s3 = ["<p><name>C</name></p>".toBytes()].toStream();
    Person|xmldata:Error r3 = xmldata:parseStream(s3);
    show(r3);
    // @output E: required field 'age' not present in XML
    stream<byte[], error?> s4 = new (new FailingBlocks());
    Person|xmldata:Error r4 = xmldata:parseStream(s4);
    show(r4 is xmldata:Error);
    // @output true
    stream<byte[], error?> s5 = new (new FailingAfterDocument());
    Person|xmldata:Error r5 = xmldata:parseStream(s5);
    show(r5);
    // @output E: stream failure

    Person|xmldata:Error latin1 = xmldata:parseString("<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?><p><name>D</name><age>4</age></p>");
    show(latin1);
    // @output {"name":"D","age":4}

    string prefix = "failed to parse xml: ";
    foreach string input in ["<p></p></q>", "</p>"] {
        Person|xmldata:Error unmatched = xmldata:parseString(input);
        show(unmatched is xmldata:Error && unmatched.message().substring(0, prefix.length()) == prefix);
    }
    // @output true
    // @output true

    JsonValue|xmldata:Error rootText = xmldata:parseString("<r>hello</r>", {textFieldName: "value"});
    show(rootText);
    // @output E: failed to parse xml
}
