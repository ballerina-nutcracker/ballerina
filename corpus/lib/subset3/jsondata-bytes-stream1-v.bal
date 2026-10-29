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

import ballerina/data.jsondata;
import ballerina/io;

type Person record {|
    string name;
    int age;
|};

type Point [int, int];

class FailingBlocks {
    int count = 0;

    public isolated function next() returns record {|byte[] value;|}|error? {
        self.count += 1;
        if self.count == 1 {
            return {value: "[1, ".toBytes()};
        }
        return error("stream failure");
    }
}

function chunk(byte[] bytes, int 'start, int end) returns byte[] {
    byte[] result = [];
    foreach int i in 'start ..< end {
        result.push(bytes[i]);
    }
    return result;
}

function show(anydata|error v) {
    if v is error {
        io:println("error: ", v.message());
    } else {
        io:println(v);
    }
}

public function main() {
    byte[] personBytes = string `{"name":"Ké","age":40}`.toBytes();
    Person|error b1 = jsondata:parseBytes(personBytes);
    show(b1);
    // @output {"name":"Ké","age":40}
    Point|error b2 = jsondata:parseBytes("[3, 4]".toBytes());
    show(b2);
    // @output [3,4]
    json|error b3 = jsondata:parseBytes([]);
    show(b3);
    // @output error: 'empty JSON document' at line: '1' column: '1'
    int|error b4 = jsondata:parseBytes("\"7\"".toBytes());
    show(b4);
    // @output error: incompatible expected type 'int' for value '7'
    map<string>|error b5 = jsondata:parseBytes(string `{"emoji":"😀"}`.toBytes());
    show(b5);
    // @output {"emoji":"😀"}

    stream<byte[], error?> s1 = [chunk(personBytes, 0, 11), chunk(personBytes, 11, personBytes.length())].toStream();
    Person|error r1 = trap jsondata:parseStream(s1, {}, Person);
    show(r1);
    // @output {"name":"Ké","age":40}
    stream<byte[], error?> s2 = ["[1,".toBytes(), " 2".toBytes(), "]".toBytes()].toStream();
    Point|error r2 = trap jsondata:parseStream(s2, {}, Point);
    show(r2);
    // @output [1,2]
    stream<byte[], error?> s3 = ["{\"name\":\"X\"}".toBytes()].toStream();
    Person|error r3 = trap jsondata:parseStream(s3, {}, Person);
    show(r3);
    // @output error: required field 'age' not present in JSON
    stream<byte[], error?> s4 = ["{\"a\" 1}".toBytes()].toStream();
    json|error r4 = trap jsondata:parseStream(s4, {}, json);
    show(r4);
    // @output error: 'expected ':'' at line: '1' column: '6'
    stream<byte[], error?> s5 = new (new FailingBlocks());
    json|error r5 = trap jsondata:parseStream(s5, {}, json);
    show(r5);
    // @output error: stream failure
    byte[][] noBlocks = [];
    stream<byte[], error?> s6 = noBlocks.toStream();
    json|error r6 = trap jsondata:parseStream(s6, {}, json);
    show(r6);
    // @output error: 'empty JSON document' at line: '1' column: '1'
    stream<byte[], error?> s7 = [[1, 2, 3].map(n => <byte>(n + 48))].toStream();
    int|error r7 = trap jsondata:parseStream(s7, {}, int);
    show(r7);
    // @output 123
}
