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

type Item record {|
    string sku;
    int qty;
|};

public function main() returns error? {
    avro:Schema intArray = check new (string `{"type": "array", "items": "int"}`);
    int[] numbers = check intArray.fromAvro(check intArray.toAvro([1, 2, 3]));
    io:println(numbers.length()); // @output 3
    io:println(numbers[2]); // @output 3

    // An empty array is a single zero-count block.
    byte[] emptyBytes = check intArray.toAvro([]);
    io:println(emptyBytes.length()); // @output 1
    int[] empty = check intArray.fromAvro(emptyBytes);
    io:println(empty.length()); // @output 0

    avro:Schema strArray = check new (string `{"type": "array", "items": "string"}`);
    string[] words = check strArray.fromAvro(check strArray.toAvro(["a", "bb"]));
    io:println(words[1]); // @output bb

    // Arrays of records.
    avro:Schema itemArray = check new (string `{
        "type": "array",
        "items": {"type": "record", "name": "Item",
                  "fields": [{"name": "sku", "type": "string"},
                             {"name": "qty", "type": "int"}]}}`);
    Item[] items = [{sku: "a", qty: 1}, {sku: "b", qty: 2}];
    Item[] backItems = check itemArray.fromAvro(check itemArray.toAvro(items));
    io:println(backItems.length()); // @output 2
    io:println(backItems[1].sku); // @output b
    io:println(backItems[0].qty); // @output 1

    // Nested arrays.
    avro:Schema matrixSchema = check new (string
        `{"type": "array", "items": {"type": "array", "items": "long"}}`);
    int[][] matrix = [[1, 2], [3]];
    int[][] backMatrix = check matrixSchema.fromAvro(check matrixSchema.toAvro(matrix));
    io:println(backMatrix[0][1]); // @output 2
    io:println(backMatrix[1].length()); // @output 1

    // A payload larger than the codec's initial buffer.
    int[] big = [];
    int i = 0;
    while i < 1000 {
        big.push(i);
        i += 1;
    }
    int[] backBig = check intArray.fromAvro(check intArray.toAvro(big));
    io:println(backBig.length()); // @output 1000
    io:println(backBig[999]); // @output 999

    // A non-list value cannot fill an array schema.
    byte[]|avro:Error notAList = intArray.toAvro(5);
    io:println(notAList is avro:Error); // @output true
}
