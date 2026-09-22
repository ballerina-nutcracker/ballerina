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

import ballerina/avro;
import ballerina/io;

type Address record {|
    string city;
    string country;
|};

type Person record {|
    string name;
    int age;
    float score;
    Address address;
    string[] tags;
|};

enum Colour {
    RED,
    GREEN,
    BLUE
}

public function main() returns error? {
    // Primitives.
    avro:Schema intSchema = check new (string `{"type": "int", "name": "i", "namespace": "demo"}`);
    byte[] encoded = check intSchema.toAvro(42);
    int decoded = check intSchema.fromAvro(encoded);
    io:println(decoded); // @output 42

    avro:Schema strSchema = check new (string `{"type": "string"}`);
    string text = check strSchema.fromAvro(check strSchema.toAvro("hello"));
    io:println(text); // @output hello

    avro:Schema bytesSchema = check new (string `{"type": "bytes"}`);
    byte[] blobSource = [1, 2, 3];
    byte[] blob = check bytesSchema.fromAvro(check bytesSchema.toAvro(blobSource));
    io:println(blob.length()); // @output 3

    // A record with a nested record and an array field.
    avro:Schema personSchema = check new (string `{
        "type": "record", "name": "Person", "namespace": "demo",
        "fields": [
            {"name": "name", "type": "string"},
            {"name": "age", "type": "int"},
            {"name": "score", "type": "double"},
            {"name": "address", "type": {"type": "record", "name": "Address",
                "fields": [{"name": "city", "type": "string"},
                           {"name": "country", "type": "string"}]}},
            {"name": "tags", "type": {"type": "array", "items": "string"}}
        ]}`);
    Person person = {
        name: "Ann",
        age: 30,
        score: 9.5,
        address: {city: "Colombo", country: "LK"},
        tags: ["a", "b"]
    };
    Person back = check personSchema.fromAvro(check personSchema.toAvro(person));
    io:println(back.name); // @output Ann
    io:println(back.address.city); // @output Colombo
    io:println(back.tags.length()); // @output 2
    io:println(back.score); // @output 9.5

    // Enum.
    avro:Schema colourSchema = check new (string
        `{"type": "enum", "name": "Colour", "symbols": ["RED", "GREEN", "BLUE"]}`);
    Colour colour = check colourSchema.fromAvro(check colourSchema.toAvro(GREEN));
    io:println(colour); // @output GREEN

    // Map.
    avro:Schema mapSchema = check new (string `{"type": "map", "values": "int"}`);
    map<int> counts = check mapSchema.fromAvro(check mapSchema.toAvro({"a": 1}));
    io:println(counts["a"]); // @output 1

    // Nullable union.
    avro:Schema optSchema = check new (string `["null", "string"]`);
    string? present = check optSchema.fromAvro(check optSchema.toAvro("set"));
    io:println(present); // @output set
    string? absent = check optSchema.fromAvro(check optSchema.toAvro(()));
    io:println(absent is ()); // @output true

    // Fixed.
    avro:Schema fixedSchema = check new (string
        `{"type": "fixed", "name": "Md5", "size": 4}`);
    byte[] digestSource = [9, 8, 7, 6];
    byte[] digest = check fixedSchema.fromAvro(check fixedSchema.toAvro(digestSource));
    io:println(digest[0]); // @output 9

    // An unparseable schema and a truncated payload both surface as avro:Error.
    avro:Schema|error bad = new ("not-a-schema");
    io:println(bad is avro:Error); // @output true
    int|avro:Error truncated = intSchema.fromAvro([]);
    io:println(truncated is avro:Error); // @output true
}
