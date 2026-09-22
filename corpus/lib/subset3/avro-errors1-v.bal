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

type Strict record {|
    int a;
    int b;
|};

public function main() returns error? {
    // An unparseable schema surfaces from `init` rather than panicking.
    avro:Schema|error garbage = new ("not-a-schema");
    io:println(garbage is avro:Error); // @output true
    if garbage is error {
        io:println(garbage.message()); // @output Avro schema parsing error
    }

    avro:Schema|error malformedJson = new ("{\"type\":");
    io:println(malformedJson is avro:Error); // @output true

    avro:Schema|error missingItems = new (string `{"type": "array"}`);
    io:println(missingItems is avro:Error); // @output true

    avro:Schema|error unknownName = new (string `{"type": "record", "name": "R",
        "fields": [{"name": "f", "type": "NoSuchType"}]}`);
    io:println(unknownName is avro:Error); // @output true

    avro:Schema intSchema = check new (string `{"type": "int"}`);

    // A payload that runs out mid-value.
    int|avro:Error truncated = intSchema.fromAvro([]);
    io:println(truncated is avro:Error); // @output true
    if truncated is error {
        io:println(truncated.message()); // @output Avro deserialization error
    }

    avro:Schema recordSchema = check new (string `{
        "type": "record", "name": "Strict", "fields": [
            {"name": "a", "type": "int"}, {"name": "b", "type": "int"}]}`);
    byte[] partial = [2];
    Strict|avro:Error short = recordSchema.fromAvro(partial);
    io:println(short is avro:Error); // @output true

    // A value that does not fit the schema.
    byte[]|avro:Error mismatch = intSchema.toAvro("text");
    io:println(mismatch is avro:Error); // @output true
    if mismatch is error {
        io:println(mismatch.message()); // @output Avro serialization error
    }

    avro:Schema nullSchema = check new (string `{"type": "null"}`);
    byte[]|avro:Error notNull = nullSchema.toAvro(1);
    io:println(notNull is avro:Error); // @output true

    // Each schema type rejects the values it cannot represent.
    avro:Schema boolSchema = check new (string `{"type": "boolean"}`);
    io:println(boolSchema.toAvro(1) is avro:Error); // @output true
    avro:Schema longSchema = check new (string `{"type": "long"}`);
    io:println(longSchema.toAvro("text") is avro:Error); // @output true
    io:println(longSchema.toAvro(1.5d) is avro:Error); // @output true
    avro:Schema doubleSchema = check new (string `{"type": "double"}`);
    io:println(doubleSchema.toAvro(true) is avro:Error); // @output true
    avro:Schema floatSchema = check new (string `{"type": "float"}`);
    io:println(floatSchema.toAvro(1.5d) is avro:Error); // @output true
    avro:Schema bytesSchema = check new (string `{"type": "bytes"}`);
    io:println(bytesSchema.toAvro("text") is avro:Error); // @output true
    io:println(bytesSchema.toAvro(["x"]) is avro:Error); // @output true
    io:println(bytesSchema.toAvro(["a", "b"]) is avro:Error); // @output true

    // toAvro validates against the value's own type, not just its elements:
    // an int[] is rejected even when every value fits in a byte.
    int[] inRangeInts = [1, 2, 3];
    io:println(bytesSchema.toAvro(inRangeInts) is avro:Error); // @output true
    io:println(bytesSchema.toAvro([300]) is avro:Error); // @output true
    avro:Schema enumSchema = check new (string
        `{"type": "enum", "name": "E", "symbols": ["A"]}`);
    io:println(enumSchema.toAvro(1) is avro:Error); // @output true
    avro:Schema recordSchemaTypes = check new (string
        `{"type": "record", "name": "T", "fields": [{"name": "a", "type": "int"}]}`);
    io:println(recordSchemaTypes.toAvro(1) is avro:Error); // @output true
    io:println(nullSchema.toAvro("text") is avro:Error); // @output true

    // A float does not narrow into an integral schema, where jBallerina lets
    // Avro's Number cast truncate it.
    io:println(longSchema.toAvro(3.9) is avro:Error); // @output true
    io:println(intSchema.toAvro(3.9) is avro:Error); // @output true

    // A decoded value the target type cannot hold.
    avro:Schema strSchema = check new (string `{"type": "string"}`);
    int|avro:Error wrongTarget = strSchema.fromAvro(check strSchema.toAvro("text"));
    io:println(wrongTarget is avro:Error); // @output true

    // A record payload missing a field the target requires.
    avro:Schema onlyA = check new (string `{
        "type": "record", "name": "OnlyA", "fields": [{"name": "a", "type": "int"}]}`);
    Strict|avro:Error incomplete = onlyA.fromAvro(check onlyA.toAvro({a: 1}));
    io:println(incomplete is avro:Error); // @output true

    // A string schema takes only a string; nil is not stringified like other values.
    io:println(strSchema.toAvro(()) is avro:Error); // @output true

    // A map or array value that fails partway through still reports an error,
    // naming the offending key or index.
    avro:Schema mapIntSchema = check new (string `{"type": "map", "values": "int"}`);
    io:println(mapIntSchema.toAvro({"a": "text"}) is avro:Error); // @output true
    avro:Schema arrayIntSchema = check new (string `{"type": "array", "items": "int"}`);
    io:println(arrayIntSchema.toAvro([1, "bad"]) is avro:Error); // @output true

    // Malformed schemas are rejected at parse time with a clear cause,
    // whichever part of the schema is missing or wrong.
    io:println((new avro:Schema(string `{"type": "map"}`)) is avro:Error); // @output true
    io:println((new avro:Schema(string `{"type": "array", "items": 5}`)) is avro:Error); // @output true
    io:println((new avro:Schema(string
        `{"type": "record", "name": "R", "fields": [{"type": "int"}]}`)) is avro:Error); // @output true
    io:println((new avro:Schema(string
        `{"type": "record", "name": "R", "fields": [{"name": "f"}]}`)) is avro:Error); // @output true
    io:println((new avro:Schema(string `{"type": "fixed", "size": 4}`)) is avro:Error); // @output true
    io:println((new avro:Schema(string `{"type": "fixed", "name": "F"}`)) is avro:Error); // @output true
    io:println((new avro:Schema(string `{"type": "enum", "symbols": ["A"]}`)) is avro:Error); // @output true
    io:println((new avro:Schema(string `{"type": "record", "name": "R", "fields": [5]}`)) is avro:Error); // @output true
    io:println((new avro:Schema(string `["null", {"type": "array"}]`)) is avro:Error); // @output true
    io:println((new avro:Schema(string `{"name": "x"}`)) is avro:Error); // @output true
    io:println((new avro:Schema(string `{"type": "map", "values": 5}`)) is avro:Error); // @output true
    io:println((new avro:Schema(string `{"type": "record", "fields": []}`)) is avro:Error); // @output true
    io:println((new avro:Schema(string `{"type": "record", "name": "R"}`)) is avro:Error); // @output true
    io:println((new avro:Schema(string `{"type": "SomeUnknownName"}`)) is avro:Error); // @output true
}
