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

public function main() returns error? {
    // null carries no bytes at all.
    avro:Schema nullSchema = check new (string `{"type": "null"}`);
    byte[] nullBytes = check nullSchema.toAvro(());
    io:println(nullBytes.length()); // @output 0
    () nothing = check nullSchema.fromAvro(nullBytes);
    io:println(nothing is ()); // @output true

    avro:Schema boolSchema = check new (string `{"type": "boolean"}`);
    boolean flag = check boolSchema.fromAvro(check boolSchema.toAvro(true));
    io:println(flag); // @output true
    boolean off = check boolSchema.fromAvro(check boolSchema.toAvro(false));
    io:println(off); // @output false

    avro:Schema intSchema = check new (string `{"type": "int"}`);
    io:println(check intSchema.fromAvro(check intSchema.toAvro(0))); // @output 0
    io:println(check intSchema.fromAvro(check intSchema.toAvro(-7))); // @output -7
    io:println(check intSchema.fromAvro(check intSchema.toAvro(2147483647))); // @output 2147483647
    io:println(check intSchema.fromAvro(check intSchema.toAvro(-2147483648))); // @output -2147483648

    // An int wider than 32 bits wraps, matching jBallerina's intValue().
    int wrapped = check intSchema.fromAvro(check intSchema.toAvro(2147483648));
    io:println(wrapped); // @output -2147483648

    avro:Schema longSchema = check new (string `{"type": "long"}`);
    int big = check longSchema.fromAvro(check longSchema.toAvro(9007199254740993));
    io:println(big); // @output 9007199254740993

    // float is single precision on the wire, so the value comes back rounded.
    avro:Schema floatSchema = check new (string `{"type": "float"}`);
    float single = check floatSchema.fromAvro(check floatSchema.toAvro(0.5));
    io:println(single); // @output 0.5

    avro:Schema doubleSchema = check new (string `{"type": "double"}`);
    float wide = check doubleSchema.fromAvro(check doubleSchema.toAvro(1.2345678901234567));
    io:println(wide); // @output 1.2345678901234567

    // decimal reaches only the double schema, where it keeps full precision.
    float fromDecimal = check doubleSchema.fromAvro(check doubleSchema.toAvro(1.2345678901234567d));
    io:println(fromDecimal); // @output 1.2345678901234567

    avro:Schema strSchema = check new (string `{"type": "string"}`);
    io:println(check strSchema.fromAvro(check strSchema.toAvro(""))); // @output
    string unicode = check strSchema.fromAvro(check strSchema.toAvro("héllo ✓"));
    io:println(unicode); // @output héllo ✓

    avro:Schema bytesSchema = check new (string `{"type": "bytes"}`);
    byte[] emptySource = [];
    byte[] empty = check bytesSchema.fromAvro(check bytesSchema.toAvro(emptySource));
    io:println(empty.length()); // @output 0
    byte[] blobSource = [0, 127, 255];
    byte[] blob = check bytesSchema.fromAvro(check bytesSchema.toAvro(blobSource));
    io:println(blob[2]); // @output 255

    // An int widens into either floating-point schema.
    float widened = check floatSchema.fromAvro(check floatSchema.toAvro(5));
    io:println(widened); // @output 5.0
    float promoted = check doubleSchema.fromAvro(check doubleSchema.toAvro(7));
    io:println(promoted); // @output 7.0

    // A float does not narrow into an integral schema — jBallerina truncates it
    // by way of Avro's Number cast, with no conversion of its own.
    io:println(intSchema.toAvro(3.9) is avro:Error); // @output true
    io:println(longSchema.toAvro(3.9) is avro:Error); // @output true

    // A string schema stringifies whatever it is given.
    io:println(check strSchema.fromAvro(check strSchema.toAvro(5))); // @output 5
    io:println(check strSchema.fromAvro(check strSchema.toAvro(true))); // @output true
    io:println(check strSchema.fromAvro(check strSchema.toAvro({a: 1}))); // @output {"a":1}

    // A logicalType annotation is ignored — the schema behaves as its plain
    // underlying primitive, matching jBallerina's GenericDatumReader/Writer,
    // which registers no logical-type conversions.
    avro:Schema tsSchema = check new (string `{"type": "long", "logicalType": "timestamp-millis"}`);
    int millis = check tsSchema.fromAvro(check tsSchema.toAvro(1234567890));
    io:println(millis); // @output 1234567890
}
