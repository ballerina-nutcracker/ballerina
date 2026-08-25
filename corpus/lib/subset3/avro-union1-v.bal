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

type Point record {|
    int x;
|};

public function main() returns error? {
    // Nullable primitives, at the top level.
    avro:Schema optString = check new (string `["null", "string"]`);
    string? text = check optString.fromAvro(check optString.toAvro("set"));
    io:println(text); // @output set
    string? none = check optString.fromAvro(check optString.toAvro(()));
    io:println(none is ()); // @output true

    avro:Schema optInt = check new (string `["null", "int"]`);
    int? counted = check optInt.fromAvro(check optInt.toAvro(5));
    io:println(counted); // @output 5

    avro:Schema optBool = check new (string `["null", "boolean"]`);
    boolean? flag = check optBool.fromAvro(check optBool.toAvro(true));
    io:println(flag); // @output true

    // Nullable composites.
    avro:Schema optRecord = check new (string
        `["null", {"type": "record", "name": "Point", "fields": [{"name": "x", "type": "int"}]}]`);
    Point? point = check optRecord.fromAvro(check optRecord.toAvro({x: 3}));
    if point is Point {
        io:println(point.x); // @output 3
    }
    Point? missing = check optRecord.fromAvro(check optRecord.toAvro(()));
    io:println(missing is ()); // @output true

    avro:Schema optArray = check new (string `["null", {"type": "array", "items": "int"}]`);
    int[]? numbers = check optArray.fromAvro(check optArray.toAvro([1, 2]));
    if numbers is int[] {
        io:println(numbers.length()); // @output 2
    }

    // The first branch whose Avro type is the value's natural encoding wins, so
    // an int takes the lossless long branch even though double comes first.
    avro:Schema doubleFirst = check new (string `["double", "long"]`);
    byte[] asLong = check doubleFirst.toAvro(5);
    io:println(asLong.length()); // @output 2
    int roundTrip = check doubleFirst.fromAvro(asLong);
    io:println(roundTrip); // @output 5

    // With no natural branch the widening pass runs, and a double branch claims
    // the int — the rule jBallerina applies to record fields.
    avro:Schema stringOrDouble = check new (string `["string", "double"]`);
    byte[] widened = check stringOrDouble.toAvro(5);
    io:println(widened.length()); // @output 9
    float asFloat = check stringOrDouble.fromAvro(widened);
    io:println(asFloat); // @output 5.0

    // The widening pass only claims what its encoder can actually write, so an
    // integral branch never takes a float.
    avro:Schema longOnly = check new (string `["long"]`);
    io:println(longOnly.toAvro(1.5) is avro:Error); // @output true

    // A decimal reaches a double branch through the widening pass.
    avro:Schema decimalTarget = check new (string `["string", "double"]`);
    float fromDecimal = check decimalTarget.fromAvro(check decimalTarget.toAvro(2.5d));
    io:println(fromDecimal); // @output 2.5

    // Declared order decides between two branches that both fit.
    avro:Schema intFirst = check new (string `["int", "string"]`);
    io:println((check intFirst.toAvro(5)).length()); // @output 2
    avro:Schema stringFirst = check new (string `["string", "int"]`);
    io:println((check stringFirst.toAvro("x")).length()); // @output 3

    // Unions inside a record field follow the same rule.
    avro:Schema fieldUnion = check new (string `{
        "type": "record", "name": "Holder", "fields": [
            {"name": "value", "type": ["null", "string"]}]}`);
    map<string?> held = check fieldUnion.fromAvro(check fieldUnion.toAvro({value: "in"}));
    io:println(held["value"]); // @output in

    // A value no branch accepts is a serialization error.
    avro:Schema noMatch = check new (string `["null", "boolean"]`);
    byte[]|avro:Error rejected = noMatch.toAvro("text");
    io:println(rejected is avro:Error); // @output true

    // A branch can be selected by type and still fail deeper — here the
    // record branch matches because the value is a mapping, but the record
    // itself is missing a required field.
    avro:Schema nullableRecord = check new (string `["null", {
        "type": "record", "name": "P",
        "fields": [{"name": "a", "type": "int"}, {"name": "b", "type": "int"}]}]`);
    io:println(nullableRecord.toAvro({a: 1}) is avro:Error); // @output true

    // With no integral branch at all, an int still widens into a float branch,
    // the same way it already widens into a double branch.
    avro:Schema stringOrFloat = check new (string `["string", "float"]`);
    byte[] widenedToFloat = check stringOrFloat.toAvro(5);
    io:println(widenedToFloat.length()); // @output 5
    float floatFromInt = check stringOrFloat.fromAvro(widenedToFloat);
    io:println(floatFromInt); // @output 5.0

    // Avro permits a union to combine a bytes/fixed branch with an array
    // branch — they're different kinds — even though a Ballerina value
    // reaches both as *values.List: a byte[] value always takes the bytes
    // branch, an int[] value always takes the array branch, in either
    // declared order.
    avro:Schema bytesThenArray = check new (string
        `["bytes", {"type": "array", "items": "int"}]`);
    avro:Schema arrayThenBytes = check new (string
        `[{"type": "array", "items": "int"}, "bytes"]`);
    byte[] rawBytes = [1, 2, 3];
    int[] rawInts = [1, 2, 3];
    io:println((check bytesThenArray.toAvro(rawBytes)).length()); // @output 5
    io:println((check arrayThenBytes.toAvro(rawBytes)).length()); // @output 5
    io:println((check bytesThenArray.toAvro(rawInts)).length()); // @output 6
    io:println((check arrayThenBytes.toAvro(rawInts)).length()); // @output 6

    // The same holds for a union combining a record branch with a map
    // branch: a record value always takes the record branch, a map<int>
    // value always takes the map branch, in either declared order.
    avro:Schema recordThenMap = check new (string
        `[{"type": "record", "name": "Point", "fields": [{"name": "x", "type": "int"}]}, {"type": "map", "values": "int"}]`);
    avro:Schema mapThenRecord = check new (string
        `[{"type": "map", "values": "int"}, {"type": "record", "name": "Point", "fields": [{"name": "x", "type": "int"}]}]`);
    Point onePoint = {x: 1};
    map<int> oneMap = {x: 1};
    io:println((check recordThenMap.toAvro(onePoint)).length()); // @output 2
    io:println((check mapThenRecord.toAvro(onePoint)).length()); // @output 2
    io:println((check recordThenMap.toAvro(oneMap)).length()); // @output 6
    io:println((check mapThenRecord.toAvro(oneMap)).length()); // @output 6
}
